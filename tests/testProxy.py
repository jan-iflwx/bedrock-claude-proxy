import pytest
from anthropic import Anthropic, NotFoundError
from langchain_anthropic import ChatAnthropic

PROXY_BASE_URL = "http://localhost:3000"
PROXY_API_KEY = "test123"
PROXY_MODEL_ID = "claude-3-5-sonnet-20241022"

# --------------
# 1.HTTP直接调用
# --------------

# --------------
# 2.官方sdk调用
# --------------

# 测试配置
@pytest.fixture
def client():
    return Anthropic(base_url=PROXY_BASE_URL, api_key=PROXY_API_KEY)

@pytest.fixture
def base_message():
    return "say this is a test"

# 基础非流式请求测试
def test_basic_completion(client, base_message):
    response = client.messages.create(
        model=PROXY_MODEL_ID,
        max_tokens=1000,
        messages=[{"role": "user", "content": base_message}]
    )
    print(response)
    assert response.content[0].text is not None
    assert len(response.content[0].text) > 0
    assert response.model == PROXY_MODEL_ID
    assert response.role == "assistant"

# 流式请求测试
def test_stream_completion(client, base_message):
    stream = client.messages.create(
        model=PROXY_MODEL_ID,
        max_tokens=1000,
        messages=[{"role": "user", "content": base_message}],
        stream=True
    )
    
    collected_message = ""
    for chunk in stream:
        if chunk.type == "content_block_delta":
            collected_message += chunk.delta.text
    
    assert len(collected_message) > 0

# 测试工具使用场景
def test_tool_use(client: Anthropic):
    # 定义工具
    tools=[
        {
            "name": "get_weather",
            "description": "Get the current weather in a given location",
            "input_schema": {
                "type": "object",
                "properties": {
                    "location": {
                        "type": "string",
                        "description": "The city and state, e.g. San Francisco, CA",
                    }
                },
                "required": ["location"],
            },
        }
    ]

    messages = [
        {
            "role": "user",
            "content": "What's the weather like in San Francisco?"
        }
    ]

    response = client.messages.create(
        model=PROXY_MODEL_ID,
        messages=messages,
        tools=tools,
        max_tokens=1000
    )

    # 验证是否调用了工具
    for content in response.content:
        if content.type == 'tool_use':
            assert content.name == "get_weather"
            assert content.input["location"] == "San Francisco, CA"
        elif content.type == 'text':
            assert content.text is not None

def test_tool_use_answer(client: Anthropic):
    # 验证输出最终结果
    response = client.messages.create(
        model=PROXY_MODEL_ID,
        max_tokens=1024,
        tools=[
            {
                "name": "get_weather",
                "description": "Get the current weather in a given location",
                "input_schema": {
                    "type": "object",
                    "properties": {
                        "location": {
                            "type": "string",
                            "description": "The city and state, e.g. San Francisco, CA"
                        },
                        "unit": {
                            "type": "string",
                            "enum": ["celsius", "fahrenheit"],
                            "description": "The unit of temperature, either 'celsius' or 'fahrenheit'"
                        }
                    },
                    "required": ["location"]
                }
            }
        ],
        messages=[
            {
                "role": "user",
                "content": "What's the weather like in San Francisco?"
            },
            {
                "role": "assistant",
                "content": [
                    {
                        "type": "text",
                        "text": "<thinking>I need to use get_weather, and the user wants SF, which is likely San Francisco, CA.</thinking>"
                    },
                    {
                        "type": "tool_use",
                        "id": "toolu_01A09q90qw90lq917835lq9",
                        "name": "get_weather",
                        "input": {"location": "San Francisco, CA", "unit": "celsius"}
                    }
                ]
            },
            {
                "role": "user",
                "content": [
                    {
                        "type": "tool_result",
                        "tool_use_id": "toolu_01A09q90qw90lq917835lq9", # from the API response
                        "content": "65 degrees" # from running your tool
                    }
                ]
            }
        ]
    )

    # 验证只有回答
    for content in response.content:
        assert content.type == 'text'
        assert content.text is not None
        # 使用`pytest -v -s testProxy.py`来查看
        print(content.text)

# 测试错误场景
def test_invalid_api_key():
    invalid_client = Anthropic(base_url=PROXY_BASE_URL, api_key="invalid_key")
    response = invalid_client.messages.create(
        model=PROXY_MODEL_ID,
        max_tokens=1000,
        messages=[{"role": "user", "content": "Hello"}]
    )
    assert response.type == "error"
    assert response.error["type"] == "invalid_request_error"
    assert response.error["message"] == "invalid api key"

# 测试模型参数限制
def test_token_limit(client):
    long_message = "test " * 5000000  # 创建一个很长的消息
    response = client.messages.create(
        model=PROXY_MODEL_ID,
        max_tokens=1000,
        messages=[{"role": "user", "content": long_message}]
    )
    assert response.type == "error"
    assert response.error["type"] == "invalid_request_error"
    assert response.error["message"].find(
        "ValidationException: 1 validation error detected: Value at 'body' failed to satisfy constraint: Member must have length less than or equal to"
    ) > 0


# 测试系统提示词
def test_system_prompt(client):
    system_prompt = "You are a helpful assistant who always responds in rhyming verse."
    response = client.messages.create(
        model=PROXY_MODEL_ID,
        max_tokens=1000,
        messages=[
            {
                "role": "user",
                "content": "Tell me about the weather"
            }
        ],
        system=system_prompt
    )
    
    assert response.content[0].text is not None
    # 这里可以添加更多具体的验证逻辑来确保回答是押韵的

# 测试温度参数
@pytest.mark.skip(reason="已通过")
@pytest.mark.parametrize("temperature", [0.0, 0.5, 1.0])
def test_temperature_parameter(client, base_message, temperature):
    response = client.messages.create(
        model=PROXY_MODEL_ID,
        max_tokens=1000,
        temperature=temperature,
        messages=[{"role": "user", "content": base_message}]
    )
    
    assert response.content[0].text is not None

# 测试beta能力: computer_use
def test_computer_use(client):
    response = client.beta.messages.create(
        model=PROXY_MODEL_ID,
        max_tokens=1024,
        tools=[
            {
                "type": "computer_20241022",
                "name": "computer",
                "display_width_px": 1024,
                "display_height_px": 768,
                "display_number": 1,
            },
            {
                "type": "text_editor_20241022",
                "name": "str_replace_editor"
            },
            {
                "type": "bash_20241022",
                "name": "bash"
            }
        ],
        messages=[{"role": "user", "content": "Save a picture of a cat to my desktop."}],
        betas=["computer-use-2024-10-22"],
    )
    print(response)

# 测试beta能力：count_tokens
def test_count_tokens(client, base_message):
    #with pytest.raises(NotFoundError, match="404 page not found"):
    response = client.beta.messages.count_tokens(
        model=PROXY_MODEL_ID,
        messages=[
            {"role": "user", "content": base_message}
        ]
    )
    print(response)
    assert(response.error is None)

# 测试beta能力：prompt_cache
def test_prompt_cache(client):
    response = client.beta.prompt_caching.messages.create(
        model=PROXY_MODEL_ID,
        max_tokens=1024,
        system=[
        {
            "type": "text",
            "text": "You are an AI assistant tasked with analyzing literary works. Your goal is to provide insightful commentary on themes, characters, and writing style.\n",
        },
        {
            "type": "text",
            "text": "<the entire contents of 'Pride and Prejudice'>",
            "cache_control": {"type": "ephemeral"}
        }
        ],
        messages=[{"role": "user", "content": "Analyze the major themes in 'Pride and Prejudice'."}],
    )
    print(response)
    assert response.error is None

def test_prompt_cache_stream(client):
    stream = client.beta.prompt_caching.messages.create(
        model=PROXY_MODEL_ID,
        max_tokens=1024,
        stream=True,
        system=[
        {
            "type": "text",
            "text": "You are an AI assistant tasked with analyzing literary works. Your goal is to provide insightful commentary on themes, characters, and writing style.\n",
        },
        {
            "type": "text",
            "text": "<the entire contents of 'Pride and Prejudice'>",
            "cache_control": {"type": "ephemeral"}
        }
        ],
        messages=[{"role": "user", "content": "Analyze the major themes in 'Pride and Prejudice'."}],
    )
    collected_message = ""
    print("\n==== chunks begin ====")
    for chunk in stream:
        print(chunk)
        print(chunk.type)
        if chunk.type == "content_block_delta":
            collected_message += chunk.delta.text
    print("---- chunks end----")
    assert len(collected_message) > 0
    
# --------------
# 3.langchain调用
# --------------
@pytest.fixture
def chat_model():
    return ChatAnthropic(
        model=PROXY_MODEL_ID,
        base_url=PROXY_BASE_URL,
        api_key=PROXY_API_KEY
    )

# 简单调用测试
def testChatAnthropic(chat_model):
    response = chat_model.invoke("say this is a test")
    assert response.content is not None
