package hostexapi

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"time"
)

const (
	BaseURL   = "https://api.hostex.io/v3"
	UserAgent = "mautrix-hostex/0.1.0"
)

type Client struct {
	httpClient  *http.Client
	accessToken string
	baseURL     string
}

type APIResponse struct {
	RequestID string      `json:"request_id"`
	Data      interface{} `json:"data,omitempty"`
	ErrorCode interface{} `json:"error_code,omitempty"`
	ErrorMsg  string      `json:"error_msg,omitempty"`
}

type Property struct {
	ID                  int     `json:"id"`
	Title               string  `json:"title"`
	Address             string  `json:"address"`
	Timezone            string  `json:"timezone"`
	DefaultCheckinTime  string  `json:"default_checkin_time"`
	DefaultCheckoutTime string  `json:"default_checkout_time"`
	WifiSSID            string  `json:"wifi_ssid"`
	WifiPassword        string  `json:"wifi_password"`
	Latitude            float64 `json:"latitude"`
	Longitude           float64 `json:"longitude"`
}

type Reservation struct {
	ReservationCode string `json:"reservation_code"`
	PropertyID      int    `json:"property_id"`
	GuestName       string `json:"guest_name"`
	GuestEmail      string `json:"guest_email"`
	GuestPhone      string `json:"guest_phone"`
	CheckInDate     string `json:"check_in_date"`
	CheckOutDate    string `json:"check_out_date"`
	Status          string `json:"status"`
	ConversationID  string `json:"conversation_id"`
	ChannelType     string `json:"channel_type"`
}

type Conversation struct {
	ID            string    `json:"id"`
	ChannelType   string    `json:"channel_type"`
	LastMessageAt time.Time `json:"last_message_at"`
	PropertyTitle string    `json:"property_title"`
	CheckInDate   string    `json:"check_in_date"`
	CheckOutDate  string    `json:"check_out_date"`
	Guest         Guest     `json:"guest"`
}

type Guest struct {
	Name  string `json:"name"`
	Phone string `json:"phone"`
	Email string `json:"email"`
}

type Message struct {
	ID          string      `json:"id"`
	SenderRole  string      `json:"sender_role"` // "guest" or "host"
	DisplayType string      `json:"display_type"`
	Content     string      `json:"content"`
	Attachment  interface{} `json:"attachment"`
	CreatedAt   time.Time   `json:"created_at"`
}

func NewClient(accessToken string) *Client {
	return &Client{
		httpClient: &http.Client{
			Timeout: 10 * time.Second, // Reduced timeout to catch hanging requests
		},
		accessToken: accessToken,
		baseURL:     BaseURL,
	}
}

func (c *Client) doRequest(ctx context.Context, method, endpoint string, body interface{}) (*APIResponse, error) {
	var reqBody []byte
	var err error

	if body != nil {
		reqBody, err = json.Marshal(body)
		if err != nil {
			return nil, fmt.Errorf("failed to marshal request body: %w", err)
		}
	}

	req, err := http.NewRequestWithContext(ctx, method, c.baseURL+endpoint, bytes.NewReader(reqBody))
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("User-Agent", UserAgent)
	req.Header.Set("Hostex-Access-Token", c.accessToken)

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to execute request: %w", err)
	}
	defer resp.Body.Close()

	var apiResp APIResponse
	if err := json.NewDecoder(resp.Body).Decode(&apiResp); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	// Check if there's actually an error (error_code != 200)
	if apiResp.ErrorCode != nil && apiResp.ErrorCode != "" {
		// Convert error code to check if it's not 200 (success)
		errorCodeStr := fmt.Sprintf("%v", apiResp.ErrorCode)
		if errorCodeStr != "200" {
			return &apiResp, fmt.Errorf("API error %v: %s", apiResp.ErrorCode, apiResp.ErrorMsg)
		}
	}

	return &apiResp, nil
}

type PropertiesResponse struct {
	Properties []Property `json:"properties"`
	Total      int        `json:"total"`
}

type ConversationsResponse struct {
	Conversations []Conversation `json:"conversations"`
}

type ReservationsResponse struct {
	Reservations []Reservation `json:"reservations"`
}

type ConversationDetails struct {
	ID          string     `json:"id"`
	ChannelType string     `json:"channel_type"`
	Guest       Guest      `json:"guest"`
	Activities  []Activity `json:"activities"`
	Note        *string    `json:"note"`
	Messages    []Message  `json:"messages"`
}

type Activity struct {
	ActivityType    string           `json:"activity_type"`
	ReservationCode *string          `json:"reservation_code"`
	CheckInDate     string           `json:"check_in_date"`
	CheckOutDate    string           `json:"check_out_date"`
	ListingID       *string          `json:"listing_id"`
	Property        ActivityProperty `json:"property"`
}

type ActivityProperty struct {
	ID       int     `json:"id"`
	Title    string  `json:"title"`
	CoverURL string  `json:"cover_url"`
	RoomType *string `json:"room_type"`
}

type ConversationDetailsResponse struct {
	ConversationDetails ConversationDetails `json:"data"`
}

func (c *Client) GetProperties(ctx context.Context) ([]Property, error) {
	resp, err := c.doRequest(ctx, "GET", "/properties", nil)
	if err != nil {
		return nil, err
	}

	// Marshal the interface{} back to JSON, then unmarshal to our struct
	dataBytes, err := json.Marshal(resp.Data)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal response data: %w", err)
	}

	var propertiesResp PropertiesResponse
	if err := json.Unmarshal(dataBytes, &propertiesResp); err != nil {
		return nil, fmt.Errorf("failed to unmarshal properties response: %w", err)
	}

	return propertiesResp.Properties, nil
}

func (c *Client) GetReservations(ctx context.Context, propertyID string) ([]Reservation, error) {
	endpoint := "/reservations?offset=0&limit=50"
	if propertyID != "" {
		endpoint += "&property_id=" + propertyID
	}

	resp, err := c.doRequest(ctx, "GET", endpoint, nil)
	if err != nil {
		return nil, err
	}

	// Marshal the interface{} back to JSON, then unmarshal to our struct
	dataBytes, err := json.Marshal(resp.Data)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal response data: %w", err)
	}

	var reservationsResp ReservationsResponse
	if err := json.Unmarshal(dataBytes, &reservationsResp); err != nil {
		return nil, fmt.Errorf("failed to unmarshal reservations response: %w", err)
	}

	return reservationsResp.Reservations, nil
}

func (c *Client) GetConversations(ctx context.Context) ([]Conversation, error) {
	// Conversations API requires offset parameter
	resp, err := c.doRequest(ctx, "GET", "/conversations?offset=0&limit=50", nil)
	if err != nil {
		return nil, err
	}

	// Marshal the interface{} back to JSON, then unmarshal to our struct
	dataBytes, err := json.Marshal(resp.Data)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal response data: %w", err)
	}

	var conversationsResp ConversationsResponse
	if err := json.Unmarshal(dataBytes, &conversationsResp); err != nil {
		return nil, fmt.Errorf("failed to unmarshal conversations response: %w", err)
	}

	return conversationsResp.Conversations, nil
}

func (c *Client) GetConversationDetails(ctx context.Context, conversationID string) (*ConversationDetails, error) {
	// Debug: Log the exact API call being made
	fullURL := BaseURL + "/conversations/" + conversationID
	fmt.Printf("DEBUG: Making API call to: %s\n", fullURL)
	fmt.Printf("DEBUG: Using token: %s...\n", c.accessToken[:10])

	resp, err := c.doRequest(ctx, "GET", "/conversations/"+conversationID, nil)
	if err != nil {
		return nil, err
	}

	// Marshal the interface{} back to JSON, then unmarshal to our struct
	dataBytes, err := json.Marshal(resp.Data)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal response data: %w", err)
	}

	var detailsResp ConversationDetails
	if err := json.Unmarshal(dataBytes, &detailsResp); err != nil {
		return nil, fmt.Errorf("failed to unmarshal conversation details: %w", err)
	}

	// Debug: Log message count and latest message times
	fmt.Printf("DEBUG: Got %d messages for conversation %s\n", len(detailsResp.Messages), conversationID)
	if len(detailsResp.Messages) > 0 {
		latestMsg := detailsResp.Messages[0] // Messages are in reverse chronological order
		fmt.Printf("DEBUG: Latest message: '%s' from %s at %s\n", latestMsg.Content, latestMsg.SenderRole, latestMsg.CreatedAt.String())

		// Log the first few messages to compare with curl output
		fmt.Printf("DEBUG: First 3 messages:\n")
		for i, msg := range detailsResp.Messages {
			if i >= 3 {
				break
			}
			fmt.Printf("  %d: '%s' from %s at %s\n", i+1, msg.Content, msg.SenderRole, msg.CreatedAt.String())
		}
	}

	return &detailsResp, nil
}

// GetMessages - Messages endpoint is not available in the Hostex API
// func (c *Client) GetMessages(ctx context.Context, conversationID string) ([]Message, error) {
//     // This endpoint returns 404 - not available in current API
//     return nil, fmt.Errorf("messages endpoint not available in Hostex API")
// }

func (c *Client) SendMessage(ctx context.Context, conversationID, content string) (*Message, error) {
	return c.SendMessageWithImage(ctx, conversationID, content, "")
}

func (c *Client) SendMessageWithImage(ctx context.Context, conversationID, content, jpegBase64 string) (*Message, error) {
	payload := map[string]interface{}{}

	// Add message if provided
	if content != "" {
		payload["message"] = content
	}

	// Add image if provided
	if jpegBase64 != "" {
		payload["jpeg"] = jpegBase64
	}

	// Must have either message or image
	if content == "" && jpegBase64 == "" {
		return nil, fmt.Errorf("must provide either message content or jpeg image")
	}

	_, err := c.doRequest(ctx, "POST", "/conversations/"+conversationID, payload)
	if err != nil {
		return nil, err
	}

	// The API doesn't return message data, just success/failure
	// So we'll create a mock message object for the bridge to use
	now := time.Now()
	displayType := "Text"
	if jpegBase64 != "" {
		if content != "" {
			displayType = "TextWithImage"
		} else {
			displayType = "Image"
		}
	}

	mockMessage := &Message{
		ID:          fmt.Sprintf("sent-%d", now.Unix()),
		SenderRole:  "host",
		DisplayType: displayType,
		Content:     content,
		Attachment:  nil, // Could store image info here if needed
		CreatedAt:   now,
	}

	return mockMessage, nil
}
