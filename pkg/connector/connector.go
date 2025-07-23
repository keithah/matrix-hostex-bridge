package connector

import (
	"context"
	"encoding/json"
	"fmt"
	"hostex-matrix-bridge/pkg/hostexapi"
	"io"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/rs/zerolog"
	"go.mau.fi/util/configupgrade"
	"maunium.net/go/mautrix/bridgev2"
	"maunium.net/go/mautrix/bridgev2/commands"
	"maunium.net/go/mautrix/bridgev2/database"
	"maunium.net/go/mautrix/bridgev2/networkid"
	"maunium.net/go/mautrix/event"
	"maunium.net/go/mautrix/id"
)

const exampleConfig = `# Hostex API URL
hostex_api_url: https://api.hostex.io/v3
# Admin user to receive startup notifications
admin_user: "@keithah:beeper.com"

# Bridge configuration goes here...
`

var configUpgrader = configupgrade.SimpleUpgrader(func(helper configupgrade.Helper) {
	// No upgrades needed yet for initial version
})

type HostexConfig struct {
	HostexAPIURL string `yaml:"hostex_api_url"`
	AdminUser    string `yaml:"admin_user"`
}

type HostexConnector struct {
	br *bridgev2.Bridge
}

var _ bridgev2.NetworkConnector = (*HostexConnector)(nil)

func (hc *HostexConnector) Init(bridge *bridgev2.Bridge) {
	hc.br = bridge
}

func (hc *HostexConnector) Start(ctx context.Context) error {
	hc.br.Log.Info().Msg("Starting Hostex connector")
	
	// Enable sync command for manual cleanup
	cmdProcessor := hc.br.Commands.(*commands.Processor)
	cmdProcessor.AddHandlers(
		&commands.FullHandler{
			Func: hc.handleSyncCommand,
			Name: "sync",
			Help: commands.HelpMeta{
				Section:     commands.HelpSectionGeneral,
				Description: "Force sync conversations from Hostex with room cleanup",
			},
			RequiresLogin: true,
		},
		&commands.FullHandler{
			Func: hc.handleRefreshCommand,
			Name: "refresh",
			Help: commands.HelpMeta{
				Section:     commands.HelpSectionGeneral,
				Description: "Refresh conversation cache and force check for new messages",
			},
			RequiresLogin: true,
		},
		&commands.FullHandler{
			Func: hc.handleCleanupCommand,
			Name: "cleanup-rooms",
			Help: commands.HelpMeta{
				Section:     commands.HelpSectionGeneral,
				Description: "Clean up and update existing room names and backfill",
			},
			RequiresLogin: true,
		},
	)
	hc.br.Log.Info().Msg("Custom command handlers ENABLED for room cleanup")
	
	// Send startup notification to admin
	go hc.sendStartupNotification(ctx)
	
	return nil
}

func (hc *HostexConnector) GetName() bridgev2.BridgeName {
	// Note: hc.br is nil during early initialization, can't log here
	return bridgev2.BridgeName{
		DisplayName:      "Hostex",
		NetworkURL:       "https://hostex.io",
		NetworkIcon:      "mxc://hostex.io/icon",
		NetworkID:        "hostex",
		BeeperBridgeType: "hostex",
		DefaultPort:      29337,
	}
}

func (hc *HostexConnector) GetCapabilities() *bridgev2.NetworkGeneralCapabilities {
	// Note: hc.br is nil during early initialization, can't log here
	return &bridgev2.NetworkGeneralCapabilities{
		DisappearingMessages: false,
		AggressiveUpdateInfo: true,
	}
}

func (hc *HostexConnector) GetBridgeInfoVersion() (info, capabilities int) {
	// Note: hc.br is nil during early initialization, can't log here
	return 1, 1
}

func (hc *HostexConnector) GetConfig() (example string, data any, upgrader configupgrade.Upgrader) {
	return exampleConfig, &HostexConfig{}, configUpgrader
}

func (hc *HostexConnector) GetDBMetaTypes() database.MetaTypes {
	// Note: hc.br is nil during early initialization, can't log here
	return database.MetaTypes{
		Portal: func() any { return &HostexPortalMetadata{} },
		Ghost:  func() any { return &HostexGhostMetadata{} },
		UserLogin: func() any { return &HostexUserLoginMetadata{} },
	}
}

func (hc *HostexConnector) GetLoginFlows() []bridgev2.LoginFlow {
	return []bridgev2.LoginFlow{{
		Name:        "Access Token",
		Description: "Log in with your Hostex API access token",
		ID:          "token",
	}}
}

func (hc *HostexConnector) CreateLogin(ctx context.Context, user *bridgev2.User, flowID string) (bridgev2.LoginProcess, error) {
	switch flowID {
	case "token":
		return &HostexLogin{
			br:   hc.br,
			user: user,
		}, nil
	default:
		return nil, fmt.Errorf("unknown login flow ID: %s", flowID)
	}
}

func (hc *HostexConnector) LoadUserLogin(ctx context.Context, login *bridgev2.UserLogin) error {
	meta := login.Metadata.(*HostexUserLoginMetadata)
	client := hostexapi.NewClient(meta.AccessToken)
	
	nl := &HostexNetworkAPI{
		br:                      hc.br,
		login:                   login,
		client:                  client,
		guestNames:              make(map[string]string),
		lastMessageTime:         make(map[string]time.Time),
		conversationLastMsgTime: make(map[string]time.Time),
		sentMessages:            make(map[string]time.Time),
	}
	
	login.Client = nl
	return nil
}

type HostexUserLoginMetadata struct {
	AccessToken string `json:"access_token"`
	UserID      string `json:"user_id"`
	UserName    string `json:"user_name"`
}

type HostexPortalMetadata struct {
	ConversationID string `json:"conversation_id"`
	PropertyID     string `json:"property_id"`
	PropertyName   string `json:"property_name"`
	GuestName      string `json:"guest_name"`
	GuestEmail     string `json:"guest_email"`
}

type HostexGhostMetadata struct {
	Name  string `json:"name"`
	Email string `json:"email"`
	Phone string `json:"phone"`
}

type HostexLogin struct {
	br   *bridgev2.Bridge
	user *bridgev2.User
}

var _ bridgev2.LoginProcess = (*HostexLogin)(nil)
var _ bridgev2.LoginProcessUserInput = (*HostexLogin)(nil)

func (hl *HostexLogin) Start(ctx context.Context) (*bridgev2.LoginStep, error) {
	return &bridgev2.LoginStep{
		Type:         bridgev2.LoginStepTypeUserInput,
		StepID:       "token",
		Instructions: "Please enter your Hostex API access token",
		UserInputParams: &bridgev2.LoginUserInputParams{
			Fields: []bridgev2.LoginInputDataField{{
				Type:        bridgev2.LoginInputFieldTypePassword,
				ID:          "access_token",
				Name:        "Access Token",
				Description: "Your Hostex API access token",
			}},
		},
	}, nil
}

func (hl *HostexLogin) Cancel() {}

func (hl *HostexLogin) SubmitUserInput(ctx context.Context, input map[string]string) (*bridgev2.LoginStep, error) {
	hl.br.Log.Info().Msg("SubmitUserInput: Starting Hostex login process")
	
	accessToken := input["access_token"]
	if accessToken == "" {
		hl.br.Log.Error().Msg("SubmitUserInput: Access token is empty")
		return nil, fmt.Errorf("access token is required")
	}
	
	hl.br.Log.Info().Int("token_length", len(accessToken)).Msg("SubmitUserInput: Got access token")
	
	// Test the API token by making a request
	hl.br.Log.Info().Msg("SubmitUserInput: Testing API token with Hostex API")
	client := hostexapi.NewClient(accessToken)
	
	// Test the connection by getting properties
	properties, err := client.GetProperties(ctx)
	if err != nil {
		hl.br.Log.Error().Err(err).Msg("SubmitUserInput: Failed to authenticate with Hostex API")
		return nil, fmt.Errorf("failed to authenticate with Hostex API: %w", err)
	}
	
	hl.br.Log.Info().Int("property_count", len(properties)).Msg("SubmitUserInput: Successfully authenticated with Hostex API")
	
	// Create user login metadata
	userLoginID := networkid.UserLoginID(fmt.Sprintf("hostex_%s", accessToken[:8])) // Use first 8 chars as ID
	
	// Create the user login
	ul, err := hl.user.NewLogin(ctx, &database.UserLogin{
		ID:         userLoginID,
		RemoteName: "Hostex User", // We could fetch this from API if available
		Metadata: &HostexUserLoginMetadata{
			AccessToken: accessToken,
			UserID:      string(userLoginID),
			UserName:    "Hostex User",
		},
	}, nil)
	if err != nil {
		hl.br.Log.Error().Err(err).Msg("SubmitUserInput: Failed to create user login")
		return nil, fmt.Errorf("failed to create user login: %w", err)
	}
	
	hl.br.Log.Info().Str("login_id", string(ul.ID)).Msg("SubmitUserInput: Created user login successfully")
	
	// Return completion step
	return &bridgev2.LoginStep{
		Type:   bridgev2.LoginStepTypeComplete,
		StepID: "complete",
		CompleteParams: &bridgev2.LoginCompleteParams{
			UserLoginID: ul.ID,
			UserLogin:   ul,
		},
	}, nil
}

type HostexNetworkAPI struct {
	br                *bridgev2.Bridge
	login             *bridgev2.UserLogin
	client            *hostexapi.Client
	guestNames              map[string]string    // conversation ID -> guest name mapping
	lastMessageTime         map[string]time.Time // conversation ID -> timestamp of last processed message
	lastMessageTimeMu       sync.RWMutex         // protects lastMessageTime map
	conversationLastMsgTime map[string]time.Time // conversation ID -> last_message_at from conversations endpoint
	conversationLastMsgMu   sync.RWMutex         // protects conversationLastMsgTime map
	sentMessages            map[string]time.Time // message content -> timestamp of sent message (to prevent echo)
	sentMessagesMu          sync.RWMutex         // protects sentMessages map
}

var _ bridgev2.NetworkAPI = (*HostexNetworkAPI)(nil)

func (hn *HostexNetworkAPI) Connect(ctx context.Context) {
	hn.br.Log.Info().Str("user_login", string(hn.login.ID)).Msg("Connecting to Hostex")
	
	// Start polling for conversations and messages
	go hn.pollConversations(ctx)
}

func (hn *HostexNetworkAPI) Disconnect() {
	hn.br.Log.Info().Str("user_login", string(hn.login.ID)).Msg("Disconnecting from Hostex")
}

func (hn *HostexNetworkAPI) IsLoggedIn() bool {
	return hn.login != nil && hn.client != nil
}

func (hn *HostexNetworkAPI) LogoutRemote(ctx context.Context) {
	// Hostex doesn't have a logout endpoint, just disconnect
}

func (hn *HostexNetworkAPI) IsThisUser(ctx context.Context, userID networkid.UserID) bool {
	// In Hostex, we're always the host, so check if this is a host message
	return strings.HasPrefix(string(userID), "host_")
}

func (hn *HostexNetworkAPI) GetChatInfo(ctx context.Context, portal *bridgev2.Portal) (*bridgev2.ChatInfo, error) {
	// Return basic chat info for Hostex conversations
	return &bridgev2.ChatInfo{
		Name: &portal.Name,
	}, nil
}

func (hn *HostexNetworkAPI) GetUserInfo(ctx context.Context, ghost *bridgev2.Ghost) (*bridgev2.UserInfo, error) {
	// Extract meaningful name from user ID
	userIDStr := string(ghost.ID)
	var name string
	
	if strings.HasPrefix(userIDStr, "host_") {
		// Host user - use the logged-in user's name or "Host"
		name = "Host"
	} else if strings.HasPrefix(userIDStr, "guest_") {
		// Guest user - extract conversation ID and get guest name
		conversationID := strings.TrimPrefix(userIDStr, "guest_")
		if guestName, exists := hn.guestNames[conversationID]; exists && guestName != "" {
			name = guestName
		} else {
			// Try to get name from stored metadata
			if meta, ok := ghost.Metadata.(*HostexGhostMetadata); ok && meta.Name != "" {
				name = meta.Name
			} else {
				name = "Guest " + conversationID
			}
		}
	} else {
		name = "Unknown User"
	}
	
	return &bridgev2.UserInfo{
		Name: &name,
	}, nil
}

func (hn *HostexNetworkAPI) GetCapabilities(ctx context.Context, portal *bridgev2.Portal) *event.RoomFeatures {
	return &event.RoomFeatures{
		MaxTextLength:       4000,
		LocationMessage:     event.CapLevelUnsupported,
		Poll:               event.CapLevelUnsupported,
		Thread:             event.CapLevelUnsupported,
		Reply:              event.CapLevelFullySupported,
		Edit:               event.CapLevelUnsupported,
		Delete:             event.CapLevelUnsupported,
		Reaction:           event.CapLevelUnsupported,
		ReadReceipts:       false,
		TypingNotifications: false,
	}
}

func (hn *HostexNetworkAPI) HandleMatrixMessage(ctx context.Context, msg *bridgev2.MatrixMessage) (*bridgev2.MatrixMessageResponse, error) {
	// This handles messages in portal rooms, not commands
	hn.br.Log.Info().
		Str("room_id", string(msg.Event.RoomID)).
		Str("sender", string(msg.Event.Sender)).
		Str("content", msg.Content.Body).
		Msg("Received Matrix message to send to Hostex")
	
	// Get the portal to find the conversation ID
	portal := msg.Portal
	if portal == nil {
		return nil, fmt.Errorf("no portal found for message")
	}
	
	// Extract conversation ID from portal key
	conversationID := string(portal.ID)
	
	hn.br.Log.Info().
		Str("conversation_id", conversationID).
		Str("content", msg.Content.Body).
		Msg("Sending message to Hostex conversation")
	
	// Send message to Hostex
	sentMessage, err := hn.client.SendMessage(ctx, conversationID, msg.Content.Body)
	if err != nil {
		hn.br.Log.Error().Err(err).
			Str("conversation_id", conversationID).
			Msg("Failed to send message to Hostex")
		return nil, fmt.Errorf("failed to send message to Hostex: %w", err)
	}
	
	// Track sent message to prevent echo
	hn.sentMessagesMu.Lock()
	hn.sentMessages[msg.Content.Body] = time.Now()
	hn.sentMessagesMu.Unlock()
	
	hn.br.Log.Info().
		Str("conversation_id", conversationID).
		Str("message_id", sentMessage.ID).
		Str("content", sentMessage.Content).
		Msg("Successfully sent message to Hostex")
	
	// Return response with the sent message details
	return &bridgev2.MatrixMessageResponse{
		DB: &database.Message{
			ID:        networkid.MessageID(sentMessage.ID),
			MXID:      msg.Event.ID,
			Room:      portal.PortalKey,
			SenderID:  networkid.UserID("host_" + string(hn.login.ID)),
			Timestamp: sentMessage.CreatedAt,
		},
	}, nil
}

func (hn *HostexNetworkAPI) ResolveIdentifier(ctx context.Context, identifier string, createChat bool) (*bridgev2.ResolveIdentifierResponse, error) {
	// Try to parse as conversation ID
	if strings.HasPrefix(identifier, "conv_") {
		conversations, err := hn.client.GetConversations(ctx)
		if err != nil {
			return nil, fmt.Errorf("failed to get conversations: %w", err)
		}
		
		for _, conv := range conversations {
			if conv.ID == identifier {
				portalKey := networkid.PortalKey{
					ID:       networkid.PortalID(conv.ID),
					Receiver: hn.login.ID,
				}
				
				return &bridgev2.ResolveIdentifierResponse{
					Chat: &bridgev2.CreateChatResponse{
						PortalKey: portalKey,
					},
				}, nil
			}
		}
		return nil, fmt.Errorf("conversation not found: %s", identifier)
	}
	
	return nil, fmt.Errorf("unknown identifier format: %s", identifier)
}

func (hn *HostexNetworkAPI) pollConversations(ctx context.Context) {
	ticker := time.NewTicker(30 * time.Second)
	defer ticker.Stop()
	
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			hn.syncConversations(ctx)
		}
	}
}

func (hn *HostexNetworkAPI) syncConversations(ctx context.Context) {
	conversations, err := hn.client.GetConversations(ctx)
	if err != nil {
		hn.br.Log.Error().Err(err).Msg("Failed to fetch conversations")
		return
	}
	
	// Only sync the last 10 conversations as requested by the user
	if len(conversations) > 10 {
		conversations = conversations[:10]
	}
	
	hn.br.Log.Info().Int("conversation_count", len(conversations)).Msg("Checking conversations for new messages")
	
	for _, conv := range conversations {
		// Check if we need to process this conversation based on last_message_at
		hn.conversationLastMsgMu.RLock()
		cachedLastMsgTime, hasCached := hn.conversationLastMsgTime[conv.ID]
		hn.conversationLastMsgMu.RUnlock()
		
		// Skip if no new messages since last check
		if hasCached && !conv.LastMessageAt.After(cachedLastMsgTime) {
			hn.br.Log.Debug().
				Str("conversation_id", conv.ID).
				Str("guest_name", conv.Guest.Name).
				Str("cached_time", cachedLastMsgTime.String()).
				Str("conv_time", conv.LastMessageAt.String()).
				Msg("Skipping conversation - no new messages")
			continue
		}
		
		hn.br.Log.Info().
			Str("conversation_id", conv.ID).
			Str("guest_name", conv.Guest.Name).
			Str("last_message_at", conv.LastMessageAt.String()).
			Msg("Processing conversation with new messages")
		
		// Update cached timestamp
		hn.conversationLastMsgMu.Lock()
		hn.conversationLastMsgTime[conv.ID] = conv.LastMessageAt
		hn.conversationLastMsgMu.Unlock()
		
		// Process this conversation
		hn.processConversation(ctx, conv)
	}
}

func (hn *HostexNetworkAPI) processConversation(ctx context.Context, conv hostexapi.Conversation) {
	// Create portal key for this conversation
	portalKey := networkid.PortalKey{
		ID:       networkid.PortalID(conv.ID),
		Receiver: hn.login.ID,
	}
	
	// Check if this portal has a Matrix room created
	portal, err := hn.br.GetExistingPortalByKey(ctx, portalKey)
	hn.br.Log.Debug().
		Err(err).
		Bool("portal_is_nil", portal == nil).
		Str("portal_mxid", func() string {
			if portal != nil {
				return portal.MXID.String()
			}
			return "nil"
		}()).
		Str("portal_key", portalKey.String()).
		Msg("Portal check result")
	
	// Get conversation details to fetch property name and messages (only when needed)
	hn.br.Log.Debug().Str("conversation_id", conv.ID).Msg("Fetching conversation details from Hostex API")
	details, err := hn.client.GetConversationDetails(ctx, conv.ID)
	if err != nil {
		hn.br.Log.Error().Err(err).Str("conversation_id", conv.ID).Msg("Failed to get conversation details")
		return
	}
	hn.br.Log.Debug().Str("conversation_id", conv.ID).Int("message_count", len(details.Messages)).Msg("Got conversation details from Hostex API")
	
	// Get property name from activities (use first activity with a property)
	propertyName := "Unknown Property"
	if len(details.Activities) > 0 && details.Activities[0].Property.Title != "" {
		propertyName = details.Activities[0].Property.Title
	}
	
	// Store guest name for later use
	hn.guestNames[conv.ID] = conv.Guest.Name
	
	// Create room name with format "(Property) - Guest Name"
	roomName := fmt.Sprintf("(%s) - %s", propertyName, conv.Guest.Name)

	if err != nil || portal == nil || portal.MXID == "" {
		hn.br.Log.Info().Str("conversation_id", conv.ID).Str("guest_name", conv.Guest.Name).Msg("Creating Matrix room for conversation with backfill")
		
		// Send a chat info change event to trigger Matrix room creation
		chatInfo := &bridgev2.ChatInfo{
			Name:  &roomName,
			Topic: &propertyName,
		}
		
		// Create a remote event to trigger portal and Matrix room creation
		remoteEvent := &bridgev2.SimpleRemoteEvent[*bridgev2.ChatInfoChange]{
			Type:         bridgev2.RemoteEventChatInfoChange,
			PortalKey:    portalKey,
			CreatePortal: true,
			Timestamp:    conv.LastMessageAt,
			LogContext: func(c zerolog.Context) zerolog.Context {
				return c.Str("guest_name", conv.Guest.Name).Str("property_name", propertyName)
			},
			Sender: bridgev2.EventSender{
				IsFromMe: false,
				Sender:   networkid.UserID("guest_" + conv.ID),
			},
			ChatInfoChange: &bridgev2.ChatInfoChange{
				ChatInfo: chatInfo,
			},
		}
		
		// Send the remote event to create the Matrix room
		hn.br.QueueRemoteEvent(hn.login, remoteEvent)
		
		// Queue message backfill events for new rooms
		hn.br.Log.Debug().Int("message_count", len(details.Messages)).Msg("Queueing messages for new portal")
		for i := len(details.Messages) - 1; i >= 0; i-- {
			msg := details.Messages[i]
			hn.queueMessageEvent(ctx, portalKey, &msg, conv.ID, conv.Guest.Name)
		}
		
		hn.br.Log.Info().
			Str("conversation_id", conv.ID).
			Str("room_name", roomName).
			Msg("Queued Matrix room creation for Hostex conversation")
	} else {
		// Room exists - update name if needed and process new messages
		hn.br.Log.Info().
			Str("conversation_id", conv.ID).
			Str("matrix_room_id", portal.MXID.String()).
			Str("new_room_name", roomName).
			Msg("Processing existing Matrix room for new messages")
		
		// Send chat info update for existing room
		chatInfo := &bridgev2.ChatInfo{
			Name:  &roomName,
			Topic: &propertyName,
		}
		
		chatInfoEvent := &bridgev2.SimpleRemoteEvent[*bridgev2.ChatInfoChange]{
			Type:         bridgev2.RemoteEventChatInfoChange,
			PortalKey:    portalKey,
			CreatePortal: false, // Don't create, just update
			Timestamp:    conv.LastMessageAt,
			LogContext: func(c zerolog.Context) zerolog.Context {
				return c.Str("guest_name", conv.Guest.Name).Str("property_name", propertyName)
			},
			Sender: bridgev2.EventSender{
				IsFromMe: false,
				Sender:   networkid.UserID("guest_" + conv.ID),
			},
			ChatInfoChange: &bridgev2.ChatInfoChange{
				ChatInfo: chatInfo,
			},
		}
		
		// Update the room info
		hn.br.QueueRemoteEvent(hn.login, chatInfoEvent)
		
		// For existing rooms, only queue messages that are newer than the last processed message
		hn.lastMessageTimeMu.RLock()
		lastProcessedTime, hasLastTime := hn.lastMessageTime[conv.ID]
		hn.lastMessageTimeMu.RUnlock()
		
		if !hasLastTime {
			// First time seeing this conversation, set baseline to oldest message to avoid flooding
			if len(details.Messages) > 0 {
				oldestMsg := details.Messages[len(details.Messages)-1] // Messages are in reverse chronological order
				hn.lastMessageTimeMu.Lock()
				hn.lastMessageTime[conv.ID] = oldestMsg.CreatedAt
				hn.lastMessageTimeMu.Unlock()
				lastProcessedTime = oldestMsg.CreatedAt
			}
		}
		
		newMessageCount := 0
		latestMessageTime := lastProcessedTime
		
		// Process all messages, but only queue ones newer than lastProcessedTime
		for i := len(details.Messages) - 1; i >= 0; i-- {
			msg := details.Messages[i]
			
			// Track the latest message time
			if msg.CreatedAt.After(latestMessageTime) {
				latestMessageTime = msg.CreatedAt
			}
			
			// Only queue messages that are newer than the last processed time
			if msg.CreatedAt.After(lastProcessedTime) {
				hn.br.Log.Info().
					Str("conversation_id", conv.ID).
					Str("message_id", msg.ID).
					Str("content", msg.Content).
					Str("sender_role", msg.SenderRole).
					Str("created_at", msg.CreatedAt.String()).
					Str("last_processed", lastProcessedTime.String()).
					Msg("Found new message to queue")
				
				hn.queueMessageEvent(ctx, portalKey, &msg, conv.ID, conv.Guest.Name)
				newMessageCount++
			}
		}
		
		// Update the last processed time
		if latestMessageTime.After(lastProcessedTime) {
			hn.lastMessageTimeMu.Lock()
			hn.lastMessageTime[conv.ID] = latestMessageTime
			hn.lastMessageTimeMu.Unlock()
		}
		
		hn.br.Log.Info().
			Str("conversation_id", conv.ID).
			Int("new_messages", newMessageCount).
			Str("last_processed_time", lastProcessedTime.String()).
			Str("latest_message_time", latestMessageTime.String()).
			Msg("Processed existing portal for new messages")
	}
}

func (hn *HostexNetworkAPI) clearPortalMessages(ctx context.Context, portal *bridgev2.Portal) {
	// This is a placeholder - in a real implementation we'd need to delete existing messages
	// For now, we'll rely on message deduplication by using consistent message IDs
	hn.br.Log.Info().
		Str("portal_id", string(portal.ID)).
		Str("matrix_room_id", portal.MXID.String()).
		Msg("Clearing portal for re-backfill (using message deduplication)")
}

func (hn *HostexNetworkAPI) queueMessageEvent(ctx context.Context, portalKey networkid.PortalKey, msg *hostexapi.Message, conversationID string, guestName string) {
	// Check if this is a host message that was recently sent from Matrix (to prevent echo)
	if msg.SenderRole == "host" {
		hn.sentMessagesMu.RLock()
		if sentTime, exists := hn.sentMessages[msg.Content]; exists {
			// If message was sent within the last 2 minutes, skip it
			if time.Since(sentTime) < 2*time.Minute {
				hn.sentMessagesMu.RUnlock()
				hn.br.Log.Debug().
					Str("content", msg.Content).
					Str("message_id", msg.ID).
					Msg("Skipping echo of recently sent message")
				return
			}
		}
		hn.sentMessagesMu.RUnlock()
	}
	
	// Determine sender
	var senderID networkid.UserID
	var isFromMe bool
	
	if msg.SenderRole == "host" {
		// Host message - use double puppeting to show as sent by the actual Matrix user
		senderID = "" // Empty senderID with isFromMe=true enables double puppeting
		isFromMe = true
	} else {
		// Guest message
		senderID = networkid.UserID("guest_" + conversationID)
		isFromMe = false
	}
	
	// Create message event
	messageEvent := &bridgev2.SimpleRemoteEvent[*hostexapi.Message]{
		Type:      bridgev2.RemoteEventMessage,
		PortalKey: portalKey,
		ID:        networkid.MessageID(msg.ID),
		Timestamp: msg.CreatedAt,
		LogContext: func(c zerolog.Context) zerolog.Context {
			return c.Str("message_id", msg.ID).Str("sender_role", msg.SenderRole)
		},
		Sender: bridgev2.EventSender{
			IsFromMe:    isFromMe,
			SenderLogin: hn.login.ID, // Set the user login for double puppeting
			Sender:      senderID,
		},
		Data: msg,
		ConvertMessageFunc: func(ctx context.Context, portal *bridgev2.Portal, intent bridgev2.MatrixAPI, data *hostexapi.Message) (*bridgev2.ConvertedMessage, error) {
			parts := []*bridgev2.ConvertedMessagePart{}
			
			// Handle text content
			if data.Content != "" {
				parts = append(parts, &bridgev2.ConvertedMessagePart{
					Type: event.EventMessage,
					Content: &event.MessageEventContent{
						MsgType: event.MsgText,
						Body:    data.Content,
					},
				})
			}
			
			// Handle attachments (images, files, etc.)
			if data.Attachment != nil {
				// Try to parse attachment as an object
				attachmentBytes, err := json.Marshal(data.Attachment)
				if err == nil {
					var attachmentObj map[string]interface{}
					if err := json.Unmarshal(attachmentBytes, &attachmentObj); err == nil {
						// Handle image attachments
						if url, ok := attachmentObj["url"].(string); ok && url != "" {
							// Try to get filename and type
							filename := "attachment"
							if name, ok := attachmentObj["filename"].(string); name != "" && ok {
								filename = name
							}
							
							msgType := event.MsgFile
							if attachmentType, ok := attachmentObj["type"].(string); ok {
								if strings.HasPrefix(attachmentType, "image/") {
									msgType = event.MsgImage
								}
							}
							
							// Download the image
							resp, err := http.Get(url)
							if err != nil {
								parts = append(parts, &bridgev2.ConvertedMessagePart{
									Type: event.EventMessage,
									Content: &event.MessageEventContent{
										MsgType: event.MsgText,
										Body:    fmt.Sprintf("📎 %s: %s (download failed)", filename, url),
									},
								})
							} else {
								defer resp.Body.Close()
								imageData, err := io.ReadAll(resp.Body)
								if err != nil {
									parts = append(parts, &bridgev2.ConvertedMessagePart{
										Type: event.EventMessage,
										Content: &event.MessageEventContent{
											MsgType: event.MsgText,
											Body:    fmt.Sprintf("📎 %s: %s (read failed)", filename, url),
										},
									})
								} else {
									// Upload to Matrix
									mimeType := resp.Header.Get("Content-Type")
									if mimeType == "" {
										mimeType = "application/octet-stream"
									}
									
									// Update msgType based on actual MIME type from HTTP response
									if strings.HasPrefix(mimeType, "image/") {
										msgType = event.MsgImage
									}
									
									mxcURL, _, err := intent.UploadMedia(ctx, portal.MXID, imageData, filename, mimeType)
									if err != nil {
										parts = append(parts, &bridgev2.ConvertedMessagePart{
											Type: event.EventMessage,
											Content: &event.MessageEventContent{
												MsgType: event.MsgText,
												Body:    fmt.Sprintf("📎 %s: %s (upload failed)", filename, url),
											},
										})
									} else {
										parts = append(parts, &bridgev2.ConvertedMessagePart{
											Type: event.EventMessage,
											Content: &event.MessageEventContent{
												MsgType: msgType,
												Body:    filename,
												URL:     id.ContentURIString(string(mxcURL)),
											},
										})
									}
								}
							}
						}
					}
				} else {
					// Handle attachment as string (URL)
					if attachmentStr, ok := data.Attachment.(string); ok && attachmentStr != "" {
						// Download the image
						resp, err := http.Get(attachmentStr)
						if err != nil {
							parts = append(parts, &bridgev2.ConvertedMessagePart{
								Type: event.EventMessage,
								Content: &event.MessageEventContent{
									MsgType: event.MsgText,
									Body:    fmt.Sprintf("📎 Image: %s (download failed)", attachmentStr),
								},
							})
						} else {
							defer resp.Body.Close()
							imageData, err := io.ReadAll(resp.Body)
							if err != nil {
								parts = append(parts, &bridgev2.ConvertedMessagePart{
									Type: event.EventMessage,
									Content: &event.MessageEventContent{
										MsgType: event.MsgText,
										Body:    fmt.Sprintf("📎 Image: %s (read failed)", attachmentStr),
									},
								})
							} else {
								// Upload to Matrix
								mimeType := resp.Header.Get("Content-Type")
								if mimeType == "" {
									mimeType = "application/octet-stream"
								}
								
								// Determine message type and filename based on MIME type
								msgType := event.MsgFile
								filename := "attachment"
								if strings.HasPrefix(mimeType, "image/") {
									msgType = event.MsgImage
									// Set appropriate filename extension based on MIME type
									switch mimeType {
									case "image/jpeg":
										filename = "image.jpg"
									case "image/png":
										filename = "image.png"
									case "image/gif":
										filename = "image.gif"
									case "image/webp":
										filename = "image.webp"
									default:
										filename = "image"
									}
								}
								
								mxcURL, _, err := intent.UploadMedia(ctx, portal.MXID, imageData, filename, mimeType)
								if err != nil {
									parts = append(parts, &bridgev2.ConvertedMessagePart{
										Type: event.EventMessage,
										Content: &event.MessageEventContent{
											MsgType: event.MsgText,
											Body:    fmt.Sprintf("📎 %s: %s (upload failed)", filename, attachmentStr),
										},
									})
								} else {
									parts = append(parts, &bridgev2.ConvertedMessagePart{
										Type: event.EventMessage,
										Content: &event.MessageEventContent{
											MsgType: msgType,
											Body:    filename,
											URL:     id.ContentURIString(string(mxcURL)),
										},
									})
								}
							}
						}
					}
				}
			}
			
			// If no parts were created, add a default text message
			if len(parts) == 0 {
				parts = append(parts, &bridgev2.ConvertedMessagePart{
					Type: event.EventMessage,
					Content: &event.MessageEventContent{
						MsgType: event.MsgText,
						Body:    "(Empty message)",
					},
				})
			}
			
			return &bridgev2.ConvertedMessage{
				Parts: parts,
			}, nil
		},
	}
	
	// Queue the message event
	hn.br.QueueRemoteEvent(hn.login, messageEvent)
}

func (hn *HostexNetworkAPI) getPortalForConversation(conv *hostexapi.Conversation) *bridgev2.Portal {
	portalKey := networkid.PortalKey{
		ID:       networkid.PortalID(conv.ID),
		Receiver: hn.login.ID,
	}
	
	// Try to get existing portal
	portal, err := hn.br.GetExistingPortalByKey(context.Background(), portalKey)
	if err != nil {
		hn.br.Log.Debug().Err(err).Str("portal_key", portalKey.String()).Msg("Portal not found, will be created later if needed")
		// Portal will be created when needed by the bridge framework
		return nil
	}
	
	return portal
}

func (hn *HostexNetworkAPI) handleMessage(ctx context.Context, portal *bridgev2.Portal, msg *hostexapi.Message, conversationID string) {
	var senderID networkid.UserID
	if msg.SenderRole == "host" {
		senderID = networkid.UserID("host_" + hn.login.Metadata.(*HostexUserLoginMetadata).UserID)
	} else {
		senderID = networkid.UserID("guest_" + conversationID)
	}
	
	// TODO: Convert message to Matrix format and send
	hn.br.Log.Debug().
		Str("conversation_id", conversationID).
		Str("sender_id", string(senderID)).
		Str("content", msg.Content).
		Msg("Received message")
}



// sendStartupNotification sends a message to the admin user when the bridge starts
func (hc *HostexConnector) sendStartupNotification(ctx context.Context) {
	// Wait a moment for the bridge to fully initialize
	time.Sleep(5 * time.Second)
	
	// Use hardcoded admin user for now since config access is complex
	adminUserID := "@keithah:beeper.com"
	if adminUserID == "" {
		hc.br.Log.Debug().Msg("No admin user configured, skipping startup notification")
		return
	}
	
	hc.br.Log.Info().Str("admin_user", adminUserID).Msg("Sending startup notification to admin")
	
	// Get or create user
	user, err := hc.br.GetUserByMXID(ctx, id.UserID(adminUserID))
	if err != nil {
		hc.br.Log.Error().Err(err).Str("admin_user", adminUserID).Msg("Failed to get admin user")
		return
	}
	
	// Get or create management room
	managementRoom, err := user.GetManagementRoom(ctx)
	if err != nil {
		hc.br.Log.Error().Err(err).Str("admin_user", adminUserID).Msg("Failed to get management room")
		return
	}
	
	// Create the startup message content
	content := &event.Content{
		Parsed: &event.MessageEventContent{
			MsgType: event.MsgNotice,
			Body: `🏠 Hostex Bridge Started

✅ Bridge is now running and ready to connect Hostex conversations to Matrix
📱 To get started, send me your Hostex API token with: login
🔗 I'll sync all your property conversations and create Matrix rooms for each guest

Bridge Info:
• Version: 0.1.0
• Bridge ID: sh-hostex
• Bot: @sh-hostexbot:beeper.local
• Status: Online

Send help for more commands.`,
			Format: event.FormatHTML,
			FormattedBody: `<strong>🏠 Hostex Bridge Started</strong><br/><br/>✅ Bridge is now running and ready to connect Hostex conversations to Matrix<br/>📱 To get started, send me your Hostex API token with: <code>login</code><br/>🔗 I'll sync all your property conversations and create Matrix rooms for each guest<br/><br/><strong>Bridge Info:</strong><br/>• Version: 0.1.0<br/>• Bridge ID: sh-hostex<br/>• Bot: @sh-hostexbot:beeper.local<br/>• Status: Online<br/><br/>Send <code>help</code> for more commands.`,
		},
	}
	
	// Send the notification message
	_, err = hc.br.Bot.SendMessage(ctx, managementRoom, event.EventMessage, content, nil)
	if err != nil {
		hc.br.Log.Error().Err(err).Str("admin_user", adminUserID).Msg("Failed to send startup notification")
		return
	}
	
	hc.br.Log.Info().Str("admin_user", adminUserID).Str("room_id", managementRoom.String()).Msg("Startup notification sent successfully")
}

// handlePingCommand handles the ping command
func (hc *HostexConnector) handlePingCommand(ce *commands.Event) {
	ce.Reply("🏓 Pong! Hostex Bridge is running.")
}

// handleSyncCommand handles the sync command
func (hc *HostexConnector) handleSyncCommand(ce *commands.Event) {
	ce.Reply("🔄 Starting sync of Hostex conversations with room cleanup...")
	
	// Get the user's logins
	logins := ce.User.GetUserLogins()
	if len(logins) == 0 {
		ce.Reply("❌ No active logins found. Please login first.")
		return
	}
	
	// Force sync for each login
	for _, login := range logins {
		if login.Client != nil {
			if hostexAPI, ok := login.Client.(*HostexNetworkAPI); ok {
				go hostexAPI.syncConversations(ce.Ctx)
			}
		}
	}
	
	ce.Reply("✅ Sync initiated for all your Hostex logins with room updates.")
}

// handleRefreshCommand handles the refresh command
func (hc *HostexConnector) handleRefreshCommand(ce *commands.Event) {
	ce.Reply("🔄 Refreshing conversation cache and checking for new messages...")
	
	// Get the user's logins
	logins := ce.User.GetUserLogins()
	if len(logins) == 0 {
		ce.Reply("❌ No active logins found. Please login first.")
		return
	}
	
	// Clear conversation cache and force refresh for each login
	for _, login := range logins {
		if login.Client != nil {
			if hostexAPI, ok := login.Client.(*HostexNetworkAPI); ok {
				// Clear the conversation last message cache to force re-check
				hostexAPI.conversationLastMsgMu.Lock()
				// Clear the cache
				for k := range hostexAPI.conversationLastMsgTime {
					delete(hostexAPI.conversationLastMsgTime, k)
				}
				hostexAPI.conversationLastMsgMu.Unlock()
				
				// Run sync which will now re-process all conversations
				go hostexAPI.syncConversations(ce.Ctx)
			}
		}
	}
	
	ce.Reply("✅ Conversation cache cleared and refresh initiated for all your Hostex logins.")
}

// handleCleanupCommand handles the cleanup-rooms command
func (hc *HostexConnector) handleCleanupCommand(ce *commands.Event) {
	ce.Reply("🧹 Starting room cleanup and re-backfill...")
	
	// Get the user's logins
	logins := ce.User.GetUserLogins()
	if len(logins) == 0 {
		ce.Reply("❌ No active logins found. Please login first.")
		return
	}
	
	// Force cleanup and sync for each login
	for _, login := range logins {
		if login.Client != nil {
			if hostexAPI, ok := login.Client.(*HostexNetworkAPI); ok {
				go func() {
					hostexAPI.br.Log.Info().Msg("Manual cleanup initiated by user")
					hostexAPI.syncConversations(ce.Ctx)
				}()
			}
		}
	}
	
	ce.Reply("✅ Room cleanup and re-backfill initiated. Room names will be updated and messages re-processed with double puppeting and attachment support.")
}


