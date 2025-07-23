package connector

import (
	"context"
	"fmt"
	"time"

	"go.mau.fi/util/configupgrade"
	"maunium.net/go/mautrix/bridgev2"
	"maunium.net/go/mautrix/bridgev2/database"
	"maunium.net/go/mautrix/bridgev2/networkid"
	"maunium.net/go/mautrix/event"
)

// Minimal connector for testing
type MinimalHostexConnector struct {
	br *bridgev2.Bridge
}

var _ bridgev2.NetworkConnector = (*MinimalHostexConnector)(nil)

func (hc *MinimalHostexConnector) Init(bridge *bridgev2.Bridge) {
	hc.br = bridge
}

func (hc *MinimalHostexConnector) Start(ctx context.Context) error {
	hc.br.Log.Info().Msg("DEBUG: Starting MINIMAL Hostex connector for testing")
	
	// Start a background monitoring goroutine to track activity
	go func() {
		defer func() {
			if r := recover(); r != nil {
				hc.br.Log.Error().Interface("panic", r).Msg("DEBUG: PANIC in background monitor goroutine!")
			}
		}()
		
		ticker := time.NewTicker(10 * time.Second)
		defer ticker.Stop()
		
		for {
			select {
			case <-ctx.Done():
				hc.br.Log.Info().Msg("DEBUG: Background monitor exiting due to context cancellation")
				return
			case <-ticker.C:
				hc.br.Log.Info().Msg("DEBUG: Bridge is still alive - background monitor tick")
			}
		}
	}()
	
	hc.br.Log.Info().Msg("DEBUG: Background monitor started")
	hc.br.Log.Info().Msg("DEBUG: Start method completed successfully")
	return nil
}

func (hc *MinimalHostexConnector) GetName() bridgev2.BridgeName {
	// Note: hc.br is nil during early initialization, can't log here
	return bridgev2.BridgeName{
		DisplayName:      "Hostex",
		NetworkURL:       "https://hostex.io",
		NetworkIcon:      "mxc://local.beeper.com/hostex-logo",  // Hostex logo from https://www.hotelminder.com/images/brand/Hostex.png
		NetworkID:        "hostex",
		BeeperBridgeType: "hostex",
		DefaultPort:      29337,
	}
}

func (hc *MinimalHostexConnector) GetCapabilities() *bridgev2.NetworkGeneralCapabilities {
	// Note: hc.br is nil during early initialization, can't log here
	return &bridgev2.NetworkGeneralCapabilities{
		DisappearingMessages: false,
		AggressiveUpdateInfo: true,
	}
}

func (hc *MinimalHostexConnector) GetBridgeInfoVersion() (info, capabilities int) {
	// Note: hc.br is nil during early initialization, can't log here
	return 1, 1
}

func (hc *MinimalHostexConnector) GetConfig() (example string, data any, upgrader configupgrade.Upgrader) {
	// Note: hc.br is nil during early initialization, can't log here
	return "# Minimal config", &struct{}{}, nil
}

// Proper metadata structs
type MinimalPortalMetadata struct {
	ID string `json:"id,omitempty"`
}

type MinimalGhostMetadata struct {
	Name string `json:"name,omitempty"`
}

type MinimalUserLoginMetadata struct {
	UserID string `json:"user_id,omitempty"`
}

func (hc *MinimalHostexConnector) GetDBMetaTypes() database.MetaTypes {
	// Note: hc.br is nil during early initialization, can't log here
	return database.MetaTypes{
		Portal:    func() any { return &MinimalPortalMetadata{} },
		Ghost:     func() any { return &MinimalGhostMetadata{} },
		UserLogin: func() any { return &MinimalUserLoginMetadata{} },
	}
}

func (hc *MinimalHostexConnector) GetLoginFlows() []bridgev2.LoginFlow {
	hc.br.Log.Info().Msg("DEBUG: MinimalHostexConnector.GetLoginFlows() called")
	defer func() {
		if r := recover(); r != nil {
			hc.br.Log.Error().Interface("panic", r).Msg("DEBUG: PANIC in GetLoginFlows!")
		}
	}()
	return []bridgev2.LoginFlow{{
		Name:        "Access Token",
		Description: "Log in with your Hostex API access token",
		ID:          "token",
	}}
}

func (hc *MinimalHostexConnector) CreateLogin(ctx context.Context, user *bridgev2.User, flowID string) (bridgev2.LoginProcess, error) {
	hc.br.Log.Info().Str("flow_id", flowID).Msg("DEBUG: CreateLogin called")
	
	switch flowID {
	case "token":
		hc.br.Log.Info().Msg("DEBUG: Creating MinimalHostexLogin for token flow")
		login := &MinimalHostexLogin{
			br:   hc.br,
			user: user,
		}
		hc.br.Log.Info().Msg("DEBUG: MinimalHostexLogin created successfully")
		return login, nil
	default:
		hc.br.Log.Error().Str("flow_id", flowID).Msg("DEBUG: Unknown flow ID")
		return nil, fmt.Errorf("unknown login flow ID: %s", flowID)
	}
}

func (hc *MinimalHostexConnector) LoadUserLogin(ctx context.Context, login *bridgev2.UserLogin) error {
	hc.br.Log.Info().Str("login_id", string(login.ID)).Msg("DEBUG: LoadUserLogin called")
	
	// Minimal implementation - create a dummy NetworkAPI
	nl := &MinimalNetworkAPI{
		br:    hc.br,
		login: login,
	}
	
	hc.br.Log.Info().Msg("DEBUG: Created MinimalNetworkAPI")
	login.Client = nl
	hc.br.Log.Info().Msg("DEBUG: LoadUserLogin completed successfully")
	return nil
}

// Minimal NetworkAPI for testing
type MinimalNetworkAPI struct {
	br    *bridgev2.Bridge
	login *bridgev2.UserLogin
}

var _ bridgev2.NetworkAPI = (*MinimalNetworkAPI)(nil)

func (hn *MinimalNetworkAPI) Connect(ctx context.Context) {
	hn.br.Log.Info().Msg("DEBUG: MinimalNetworkAPI.Connect() called")
}

func (hn *MinimalNetworkAPI) Disconnect() {
	hn.br.Log.Info().Msg("DEBUG: MinimalNetworkAPI.Disconnect() called")
}

func (hn *MinimalNetworkAPI) IsLoggedIn() bool {
	hn.br.Log.Info().Msg("DEBUG: MinimalNetworkAPI.IsLoggedIn() called")
	defer func() {
		if r := recover(); r != nil {
			hn.br.Log.Error().Interface("panic", r).Msg("DEBUG: PANIC in IsLoggedIn!")
		}
	}()
	return false // Always logged out in minimal test
}

func (hn *MinimalNetworkAPI) LogoutRemote(ctx context.Context) {
	hn.br.Log.Info().Msg("DEBUG: MinimalNetworkAPI.LogoutRemote() called")
	defer func() {
		if r := recover(); r != nil {
			hn.br.Log.Error().Interface("panic", r).Msg("DEBUG: PANIC in LogoutRemote!")
		}
	}()
	// Do nothing in minimal test
}

func (hn *MinimalNetworkAPI) IsThisUser(ctx context.Context, userID networkid.UserID) bool {
	hn.br.Log.Info().Str("user_id", string(userID)).Msg("DEBUG: MinimalNetworkAPI.IsThisUser() called")
	defer func() {
		if r := recover(); r != nil {
			hn.br.Log.Error().Interface("panic", r).Msg("DEBUG: PANIC in IsThisUser!")
		}
	}()
	return false
}

func (hn *MinimalNetworkAPI) GetChatInfo(ctx context.Context, portal *bridgev2.Portal) (*bridgev2.ChatInfo, error) {
	hn.br.Log.Info().Msg("DEBUG: MinimalNetworkAPI.GetChatInfo() called")
	defer func() {
		if r := recover(); r != nil {
			hn.br.Log.Error().Interface("panic", r).Msg("DEBUG: PANIC in GetChatInfo!")
		}
	}()
	return &bridgev2.ChatInfo{}, nil
}

func (hn *MinimalNetworkAPI) GetUserInfo(ctx context.Context, ghost *bridgev2.Ghost) (*bridgev2.UserInfo, error) {
	hn.br.Log.Info().Msg("DEBUG: MinimalNetworkAPI.GetUserInfo() called")
	defer func() {
		if r := recover(); r != nil {
			hn.br.Log.Error().Interface("panic", r).Msg("DEBUG: PANIC in GetUserInfo!")
		}
	}()
	return &bridgev2.UserInfo{}, nil
}

func (hn *MinimalNetworkAPI) GetCapabilities(ctx context.Context, portal *bridgev2.Portal) *event.RoomFeatures {
	hn.br.Log.Info().Msg("DEBUG: MinimalNetworkAPI.GetCapabilities() called")
	defer func() {
		if r := recover(); r != nil {
			hn.br.Log.Error().Interface("panic", r).Msg("DEBUG: PANIC in GetCapabilities!")
		}
	}()
	return &event.RoomFeatures{}
}

func (hn *MinimalNetworkAPI) HandleMatrixMessage(ctx context.Context, msg *bridgev2.MatrixMessage) (*bridgev2.MatrixMessageResponse, error) {
	hn.br.Log.Info().Msg("DEBUG: MinimalNetworkAPI.HandleMatrixMessage() called")
	defer func() {
		if r := recover(); r != nil {
			hn.br.Log.Error().Interface("panic", r).Msg("DEBUG: PANIC in HandleMatrixMessage!")
		}
	}()
	return &bridgev2.MatrixMessageResponse{}, nil
}

func (hn *MinimalNetworkAPI) ResolveIdentifier(ctx context.Context, identifier string, createChat bool) (*bridgev2.ResolveIdentifierResponse, error) {
	hn.br.Log.Info().Str("identifier", identifier).Bool("create_chat", createChat).Msg("DEBUG: MinimalNetworkAPI.ResolveIdentifier() called")
	defer func() {
		if r := recover(); r != nil {
			hn.br.Log.Error().Interface("panic", r).Msg("DEBUG: PANIC in ResolveIdentifier!")
		}
	}()
	return nil, fmt.Errorf("identifier resolution not supported in minimal test")
}

// Minimal Login Process
type MinimalHostexLogin struct {
	br   *bridgev2.Bridge
	user *bridgev2.User
}

var _ bridgev2.LoginProcess = (*MinimalHostexLogin)(nil)
var _ bridgev2.LoginProcessUserInput = (*MinimalHostexLogin)(nil)

func (hl *MinimalHostexLogin) Start(ctx context.Context) (*bridgev2.LoginStep, error) {
	hl.br.Log.Info().Msg("DEBUG: MinimalHostexLogin.Start() called")
	
	step := &bridgev2.LoginStep{
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
	}
	
	hl.br.Log.Info().Msg("DEBUG: LoginStep created, returning from Start()")
	return step, nil
}

func (hl *MinimalHostexLogin) Cancel() {
	hl.br.Log.Info().Msg("DEBUG: MinimalHostexLogin.Cancel() called")
}

func (hl *MinimalHostexLogin) SubmitUserInput(ctx context.Context, input map[string]string) (*bridgev2.LoginStep, error) {
	defer func() {
		if r := recover(); r != nil {
			hl.br.Log.Error().Interface("panic", r).Msg("DEBUG: PANIC in SubmitUserInput!")
		}
	}()
	
	hl.br.Log.Info().Msg("DEBUG: MinimalHostexLogin.SubmitUserInput() called - ENTRY POINT")
	hl.br.Log.Info().Interface("input", input).Msg("DEBUG: Input received")
	
	accessToken := input["access_token"]
	if accessToken == "" {
		hl.br.Log.Error().Msg("DEBUG: Access token is empty")
		return nil, fmt.Errorf("access token is required")
	}
	
	hl.br.Log.Info().Int("token_length", len(accessToken)).Msg("DEBUG: Got access token")
	hl.br.Log.Info().Msg("DEBUG: About to return test error - ALMOST DONE")
	
	// Just return a test error to verify the login flow works without crashing
	err := fmt.Errorf("MINIMAL LOGIN TEST: Got token with %d characters - login flow is working!", len(accessToken))
	hl.br.Log.Info().Err(err).Msg("DEBUG: Returning error - SUCCESS!")
	return nil, err
}