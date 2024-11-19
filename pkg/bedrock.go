package pkg

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"log"
	"os"
	"strings"

	"github.com/aws/aws-sdk-go-v2/aws"
	awsConfig "github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/credentials"
	bedrock "github.com/aws/aws-sdk-go-v2/service/bedrockruntime"
	"github.com/aws/aws-sdk-go-v2/service/bedrockruntime/types"
	"github.com/aws/aws-sdk-go-v2/service/sts"
)

type BedrockConfig struct {
	AccessKey                string            `json:"access_key"`
	SecretKey                string            `json:"secret_key"`
	Region                   string            `json:"region"`
	RoleArn                  string            `json:"role_arn"`
	RoleRegion               string            `json:"role_region"`
	AnthropicVersionMappings map[string]string `json:"anthropic_version_mappings"`
	ModelMappings            map[string]string `json:"model_mappings"`
	AnthropicDefaultModel    string            `json:"anthropic_default_model"`
	AnthropicDefaultVersion  string            `json:"anthropic_default_version"`
}

func (config *BedrockConfig) GetInvokeEndpoint(modelId string) string {
	return fmt.Sprintf("bedrock-runtime.%s.amazonaws.com/model/%s/invoke", config.Region, modelId)
}

func (config *BedrockConfig) GetInvokeStreamEndpoint(modelId string, region string) string {
	return fmt.Sprintf("bedrock-runtime.%s.amazonaws.com/model/%s/invoke-with-response-stream", region, modelId)
}

func ParseMappingsFromStr(raw string) map[string]string {
	mappings := map[string]string{}
	pairs := strings.Split(raw, ",")
	// 遍历每个键值对
	for _, pair := range pairs {
		// 以等号分割键和值
		kv := strings.Split(pair, "=")
		if len(kv) == 2 {
			key := strings.TrimSpace(kv[0])
			value := strings.TrimSpace(kv[1])
			mappings[key] = value
		}
	}

	return mappings
}

func LoadBedrockConfigWithEnv() *BedrockConfig {
	return &BedrockConfig{
		AccessKey:                os.Getenv("AWS_BEDROCK_ACCESS_KEY"),
		SecretKey:                os.Getenv("AWS_BEDROCK_SECRET_KEY"),
		Region:                   os.Getenv("AWS_BEDROCK_REGION"),
		RoleArn:                  os.Getenv("AWS_BEDROCK_ROLE_ARN"),
		RoleRegion:               os.Getenv("AWS_BEDROCK_ROLE_REGION"),
		ModelMappings:            ParseMappingsFromStr(os.Getenv("AWS_BEDROCK_MODEL_MAPPINGS")),
		AnthropicVersionMappings: ParseMappingsFromStr(os.Getenv("AWS_BEDROCK_ANTHROPIC_VERSION_MAPPINGS")),
		AnthropicDefaultModel:    os.Getenv("AWS_BEDROCK_ANTHROPIC_DEFAULT_MODEL"),
		AnthropicDefaultVersion:  os.Getenv("AWS_BEDROCK_ANTHROPIC_DEFAULT_VERSION"),
	}
}

type BedrockClient struct {
	config *BedrockConfig
	client *bedrock.Client
}

type ClaudeTextCompletionRequest struct {
	Prompt            string   `json:"prompt,omitempty"`
	MaxTokensToSample int      `json:"max_tokens_to_sample,omitempty"`
	Temperature       float64  `json:"temperature,omitempty"`
	StopSequences     []string `json:"stop_sequences,omitempty"`
	TopP              float64  `json:"top_p,omitempty"`
	TopK              int      `json:"top_k,omitempty"`
	Stream            bool     `json:"-"`
	Model             string   `json:"-"`
}

func (request *ClaudeTextCompletionRequest) UnmarshalJSON(data []byte) error {
	type Alias ClaudeTextCompletionRequest
	tmp := &struct {
		*Alias

		Stream bool   `json:"stream"`
		Model  string `json:"model"`
	}{
		Stream: false,
		Alias:  (*Alias)(request),
	}
	if err := json.Unmarshal(data, &tmp); err != nil {
		return err
	}

	request.Stream = tmp.Stream
	request.Model = tmp.Model

	//Log.Debug("ClaudeTextCompletionRequest UnmarshalJSON")
	//Log.Debug(tests.ToJSON(tmp))
	//Log.Debugf("%+v", this)

	return nil
}

type ClaudeMessageCompletionRequestContentSource struct {
	Type      string `json:"type,omitempty"`
	MediaType string `json:"media_type,omitempty"`
	Data      string `json:"data,omitempty"`
}

type ClaudeMessageCompletionRequestContent struct {
	Type      string                                       `json:"type,omitempty"`
	Name      string                                       `json:"name,omitempty"`
	Id        string                                       `json:"id,omitempty"`
	Text      string                                       `json:"text,omitempty"`
	ToolUseID string                                       `json:"tool_use_id,omitempty"`
	IsError   string                                       `json:"is_error,omitempty"`
	Source    *ClaudeMessageCompletionRequestContentSource `json:"source,omitempty"`
	Content   json.RawMessage                              `json:"content,omitempty"`
}

type ClaudeMessageCompletionRequestMessage struct {
	Role    string          `json:"role,omitempty"`
	Content json.RawMessage `json:"content,omitempty"`
	Text    string          `json:"text,omitempty"`
}

type ClaudeMessageCompletionRequestMetadata struct {
	UserId string `json:"user_id,omitempty"`
}

type ClaudeMessageCompletionRequestInputSchema struct {
	Type       string                                                   `json:"type,omitempty"`
	Properties map[string]*ClaudeMessageCompletionRequestPropertiesItem `json:"properties,omitempty"`
	Required   []string                                                 `json:"required,omitempty"`
}

type ClaudeMessageCompletionRequestPropertiesItem struct {
	Type        string `json:"type,omitempty"`
	Description string `json:"description,omitempty"`
}

type ClaudeMessageCompletionRequestTools struct {
	Type        string                                     `json:"type,omitempty"`
	Name        string                                     `json:"name,omitempty"`
	Description string                                     `json:"description,omitempty"`
	InputSchema *ClaudeMessageCompletionRequestInputSchema `json:"input_schema,omitempty"`
	// add for computer use
	DisplayHeightPx int                                    `json:"display_height_px,omitempty"`
	DisplayWidthPx int                                     `json:"display_width_px,omitempty"`
	DisplayNumber int                                      `json:"display_number,omitempty"`
}

type ClaudeMessageCompletionRequest struct {
	Temperature      float64                                  `json:"temperature,omitempty"`
	StopSequences    []string                                 `json:"stop_sequences,omitempty"`
	TopP             float64                                  `json:"top_p,omitempty"`
	TopK             int                                      `json:"top_k,omitempty"`
	Stream           bool                                     `json:"-"`
	Model            string                                   `json:"-"`
	AnthropicVersion string                                   `json:"anthropic_version,omitempty"`
	AnthropicBeta    []string                                 `json:"anthropic_beta,omitempty"`
	MaxToken         int                                      `json:"max_tokens,omitempty"`
	System           json.RawMessage                          `json:"system,omitempty"`
	Messages         []*ClaudeMessageCompletionRequestMessage `json:"messages,omitempty"`
	Metadata         *ClaudeMessageCompletionRequestMetadata  `json:"-"`
	Tools            []*ClaudeMessageCompletionRequestTools   `json:"tools,omitempty"`
}

func (request *ClaudeMessageCompletionRequest) UnmarshalJSON(data []byte) error {
	type Alias ClaudeMessageCompletionRequest
	tmp := &struct {
		*Alias

		Stream   bool                                    `json:"stream"`
		Model    string                                  `json:"model"`
		Metadata *ClaudeMessageCompletionRequestMetadata `json:"metadata"`
		Tools    []*ClaudeMessageCompletionRequestTools  `json:"tools"`
	}{
		Stream: false,
		Alias:  (*Alias)(request),
	}

	if err := json.Unmarshal(data, &tmp); err != nil {
		return err
	}

	request.Metadata = tmp.Metadata
	if tmp.Tools != nil {
		request.Tools = tmp.Tools
	} else {
		request.Tools = []*ClaudeMessageCompletionRequestTools{}
	}
	if request.TopK < 0 {
		request.TopK = 0
	}
	if request.TopP < 0 {
		request.TopP = 0.0
	}
	request.Stream = tmp.Stream
	request.Model = tmp.Model

	//Log.Debug("ClaudeMessageCompletionRequest UnmarshalJSON")
	//Log.Debug(tests.ToJSON(tmp))
	//Log.Debugf("%+v", this)

	return nil
}

type ClaudeTextCompletionResponse struct {
	Completion string `json:"completion,omitempty"`
	StopReason string `json:"stop_reason,omitempty"`
	Stop       string `json:"stop,omitempty"`
	Id         string `json:"id,omitempty"`
	Model      string `json:"model,omitempty"`
}

type ClaudeMessageCompletionResponse struct {
	ClaudeMessageStop

	Id      string                       `json:"id,omitempty"`
	Model   string                       `json:"model,omitempty"`
	Type    string                       `json:"type,omitempty"`
	Role    string                       `json:"role,omitempty"`
	Content []*ClaudeMessageContentBlock `json:"content,omitempty"`
	Usage   *ClaudeMessageUsage          `json:"usage,omitempty"`
}

type ISSEDecoder interface {
	GetBytes() []byte
	GetEvent() string
	GetText() string
}

type ClaudeTextCompletionStreamEvent struct {
	Type       string `json:"type,omitempty"`
	StopReason string `json:"stop_reason,omitempty"`
	Model      string `json:"model,omitempty"`
	Completion string `json:"completion,omitempty"`
	Raw        []byte `json:"-"`
}

func (event *ClaudeTextCompletionStreamEvent) GetBytes() []byte {
	return event.Raw
}

func (event *ClaudeTextCompletionStreamEvent) GetEvent() string {
	return event.Type
}

func (event *ClaudeTextCompletionStreamEvent) GetText() string {
	return event.Completion
}

type ClaudeMessageUsage struct {
	InputTokens  int `json:"input_tokens,omitempty"`
	OutputTokens int `json:"output_tokens,omitempty"`
}

type ClaudeMessageStop struct {
	StopReason   string `json:"stop_reason,omitempty"`
	StopSequence string `json:"stop_sequence,omitempty"`
}
type ClaudeMessageInfo struct {
	ClaudeMessageStop

	Id      string              `json:"id,omitempty"`
	Type    string              `json:"type,omitempty"`
	Role    string              `json:"role,omitempty"`
	Content []string            `json:"content,omitempty"`
	Model   string              `json:"model,omitempty"`
	Usage   *ClaudeMessageUsage `json:"usage,omitempty"`
}
type ClaudeMessageContentBlock struct {
	Type  string      `json:"type,omitempty"`
	Text  string      `json:"text,omitempty"`
	Id    string      `json:"id,omitempty"`
	Name  string      `json:"name,omitempty"`
	Input interface{} `json:"input,omitempty"`
}
type ClaudeMessageDelta struct {
	ClaudeMessageStop

	Type        string `json:"type,omitempty"`
	Text        string `json:"text,omitempty"`
	PartialJson string `json:"partial_json,omitempty"`
}

type ClaudeMessageCompletionStreamEvent struct {
	Type         string                     `json:"type,omitempty"`
	Model        string                     `json:"model,omitempty"`
	Completion   string                     `json:"completion,omitempty"`
	Message      *ClaudeMessageInfo         `json:"message,omitempty"`
	Usage        *ClaudeMessageUsage        `json:"usage,omitempty"`
	Index        int                        `json:"index,omitempty"`
	ContentBlock *ClaudeMessageContentBlock `json:"content_block,omitempty"`
	Delta        *ClaudeMessageDelta        `json:"delta,omitempty"`
	Raw          []byte                     `json:"-"`
}

func (event *ClaudeMessageCompletionStreamEvent) GetBytes() []byte {
	return event.Raw
}

func (event *ClaudeMessageCompletionStreamEvent) GetEvent() string {
	return event.Type
}

func (event *ClaudeMessageCompletionStreamEvent) GetText() string {
	if event.Delta != nil {
		return event.Delta.Text
	}
	return event.Completion
}

type CompleteTextResponse struct {
	stream   bool
	Response *ClaudeTextCompletionResponse
	Events   <-chan ISSEDecoder
}

func NewStreamCompleteTextResponse(queue <-chan ISSEDecoder) *CompleteTextResponse {
	return &CompleteTextResponse{
		stream: true,
		Events: queue,
	}
}

type IStreamableResponse interface {
	IsStream() bool
	GetResponse() interface{}
	GetEvents() <-chan ISSEDecoder
}

func NewCompleteTextResponse(response *ClaudeTextCompletionResponse) *CompleteTextResponse {
	return &CompleteTextResponse{
		stream:   false,
		Response: response,
	}
}

func (response *CompleteTextResponse) IsStream() bool {
	return response.stream
}

func (response *CompleteTextResponse) GetResponse() interface{} {
	return response.Response
}

func (response *CompleteTextResponse) GetEvents() <-chan ISSEDecoder {
	return response.Events
}

type MessageCompleteResponse struct {
	stream   bool
	Response *ClaudeMessageCompletionResponse
	Events   <-chan ISSEDecoder
}

func NewStreamMessageCompleteResponse(queue <-chan ISSEDecoder) *MessageCompleteResponse {
	return &MessageCompleteResponse{
		stream: true,
		Events: queue,
	}
}

func NewMessageCompleteResponse(response *ClaudeMessageCompletionResponse) *MessageCompleteResponse {
	return &MessageCompleteResponse{
		stream:   false,
		Response: response,
	}
}

func (response *MessageCompleteResponse) IsStream() bool {
	return response.stream
}

func (response *MessageCompleteResponse) GetResponse() interface{} {
	return response.Response
}

func (response *MessageCompleteResponse) GetEvents() <-chan ISSEDecoder {
	return response.Events
}

func NewSSERaw(encoder ISSEDecoder) []byte {
	return []byte(fmt.Sprintf("event: %s\ndata: %s\n\n", encoder.GetEvent(), string(encoder.GetBytes())))
}

type ClaudeTextCompletionStreamEventList []*ClaudeTextCompletionStreamEvent

func (eventList *ClaudeTextCompletionStreamEventList) Completion() string {
	var completion string
	for _, event := range *eventList {
		completion += event.Completion
	}
	return completion
}

func NewBedrockClient(config *BedrockConfig) *BedrockClient {
	staticProvider := credentials.NewStaticCredentialsProvider(config.AccessKey, config.SecretKey, "")

	cfg, err := awsConfig.LoadDefaultConfig(context.TODO(),
		awsConfig.WithRegion(config.Region),
		awsConfig.WithCredentialsProvider(staticProvider))
	if err != nil {
		log.Fatalf("unable to load SDK config, %v", err)
	}

	// if not set RoleArn
	if config.RoleArn == "" {
		return &BedrockClient{
			config: config,
			client: bedrock.NewFromConfig(cfg),
		}
	}

	// ===== assume role ======
	stsSvc := sts.NewFromConfig(cfg)

	// Assume role
	input := &sts.AssumeRoleInput{
		RoleArn:         aws.String(config.RoleArn),
		RoleSessionName: aws.String("bedrockruntime-session"),
	}
	result, err := stsSvc.AssumeRole(context.TODO(), input)
	if err != nil {
		log.Fatalf("unable to assume role, %v", err)
		return nil
	}

	// Use assumed role to create new credentials
	assumedCreds := aws.NewCredentialsCache(credentials.NewStaticCredentialsProvider(
		*result.Credentials.AccessKeyId,
		*result.Credentials.SecretAccessKey,
		*result.Credentials.SessionToken,
	))

	// Create a BedrockRuntime client using the assumed role credentials
	bedrock_cfg, err := awsConfig.LoadDefaultConfig(
		context.TODO(),
		awsConfig.WithRegion(config.RoleRegion),
		awsConfig.WithCredentialsProvider(assumedCreds),
	)
	if err != nil {
		log.Fatalf("unable to create bedrock runtime with assummed role, %v", err)
		return nil
	}

	return &BedrockClient{
		config: config,
		client: bedrock.NewFromConfig(bedrock_cfg),
	}
}

func (client *BedrockClient) CompleteText(req *ClaudeTextCompletionRequest) (IStreamableResponse, error) {
	modelId := req.Model
	mappedModel, exist := client.config.ModelMappings[modelId]
	if exist {
		modelId = mappedModel
	}
	if len(modelId) == 0 {
		modelId = client.config.AnthropicDefaultModel
	}

	if !strings.HasSuffix(req.Prompt, "Assistant:") {
		req.Prompt = fmt.Sprintf("\n\nHuman: %s\n\nAssistant:", req.Prompt)
	}
	body, err := json.Marshal(req)
	if err != nil {
		Log.Errorf("Couldn't marshal the request: ", err)
		return nil, err
	}

	if req.Stream {
		output, err := client.client.InvokeModelWithResponseStream(context.Background(), &bedrock.InvokeModelWithResponseStreamInput{
			Body:        body,
			ModelId:     aws.String(modelId),
			ContentType: aws.String("application/json"),
		})
		if err != nil {
			Log.Error(err)
			return nil, err
		}

		//Log.Debugf("Request: %+v", output)

		reader := output.GetStream()
		eventQueue := make(chan ISSEDecoder, 10)

		go func() {
			defer reader.Close()
			defer close(eventQueue)

			for event := range reader.Events() {
				switch v := event.(type) {
				case *types.ResponseStreamMemberChunk:

					//Log.Info("payload", string(v.Value.Bytes))

					var resp ClaudeTextCompletionStreamEvent
					err := json.NewDecoder(bytes.NewReader(v.Value.Bytes)).Decode(&resp)
					if err != nil {
						Log.Error(err)
						continue
					}
					resp.Raw = v.Value.Bytes
					eventQueue <- &resp

				case *types.UnknownUnionMember:
					Log.Errorf("unknown tag:", v.Tag)
					continue
				default:
					Log.Errorf("union is nil or unknown type")
					continue
				}
			}
		}()

		return NewStreamCompleteTextResponse(eventQueue), nil
	}

	output, err := client.client.InvokeModel(context.Background(), &bedrock.InvokeModelInput{
		Body:        body,
		ModelId:     aws.String(modelId),
		ContentType: aws.String("application/json"),
	})
	if err != nil {
		Log.Error(err)
		return nil, err
	}

	if output.Body != nil {
		var resp ClaudeTextCompletionResponse
		err = json.NewDecoder(bytes.NewReader(output.Body)).Decode(&resp)
		if err != nil {
			Log.Error(err)
			return nil, err
		}
		//Log.Debug(resp)

		return NewCompleteTextResponse(&resp), nil
	}

	return nil, nil
}

func (client *BedrockClient) MessageCompletion(req *ClaudeMessageCompletionRequest) (IStreamableResponse, error) {
	modelId := req.Model
	mappedModel, exist := client.config.ModelMappings[modelId]
	if exist {
		modelId = mappedModel
	}
	if len(modelId) == 0 {
		modelId = client.config.AnthropicDefaultModel
	}
	apiVersion, exist := client.config.AnthropicVersionMappings[req.AnthropicVersion]
	if exist {
		req.AnthropicVersion = apiVersion
	}
	if len(req.AnthropicVersion) == 0 {
		req.AnthropicVersion = client.config.AnthropicDefaultVersion
	}

	body, err := json.Marshal(req)
	if err != nil {
		Log.Errorf("Couldn't marshal the request: ", err)
		return nil, err
	}

	Log.Debugf("Request: %s", string(body))
	Log.Debugf("Request Model ID: %s", modelId)

	if req.Stream {
		output, err := client.client.InvokeModelWithResponseStream(context.Background(), &bedrock.InvokeModelWithResponseStreamInput{
			Body:        body,
			ModelId:     aws.String(modelId),
			ContentType: aws.String("application/json"),
		})
		if err != nil {
			Log.Error(err)
			return nil, err
		}

		reader := output.GetStream()
		eventQueue := make(chan ISSEDecoder, 10)

		go func() {
			defer reader.Close()
			defer close(eventQueue)

			for event := range reader.Events() {
				switch v := event.(type) {
				case *types.ResponseStreamMemberChunk:

					//Log.Info("payload", string(v.Value.Bytes))

					var resp ClaudeMessageCompletionStreamEvent
					err := json.NewDecoder(bytes.NewReader(v.Value.Bytes)).Decode(&resp)
					if err != nil {
						Log.Error(err)
						continue
					}
					resp.Raw = v.Value.Bytes
					eventQueue <- &resp

				case *types.UnknownUnionMember:
					Log.Errorf("unknown tag:", v.Tag)
					continue
				default:
					Log.Errorf("union is nil or unknown type")
					continue
				}
			}
		}()

		return NewStreamMessageCompleteResponse(eventQueue), nil
	}

	output, err := client.client.InvokeModel(context.Background(), &bedrock.InvokeModelInput{
		Body:        body,
		ModelId:     aws.String(modelId),
		ContentType: aws.String("application/json"),
	})
	if err != nil {
		Log.Error(err)
		return nil, err
	}

	if output.Body != nil {
		var resp ClaudeMessageCompletionResponse
		err = json.NewDecoder(bytes.NewReader(output.Body)).Decode(&resp)
		if err != nil {
			Log.Error(err)
			return nil, err
		}
		//Log.Debug(resp)

		return NewMessageCompleteResponse(&resp), nil
	}

	return nil, nil
}
