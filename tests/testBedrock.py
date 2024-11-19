body_json = {
    "anthropic_version": "bedrock-2023-05-31",
    "anthropic_beta": [
        "prompt-caching-2024-07-31"
    ],
    "max_tokens": 8192,
    "system": [
        {
            "type": "text",
            "text": "You are an AI assistant tasked with analyzing literary works. Your goal is to provide insightful commentary on themes, characters, and writing style.\n",
        },
        {
            "type": "text",
            "text": "<the entire contents of 'Pride and Prejudice'>",
            "cache_control": {"type": "ephemeral"}
        }
    ]
}
modelId = "anthropic.claude-3-5-sonnet-20241022-v2:0"

import json
body = json.dumps(body_json)
accept = "application/json"
contentType = "application/json"

import os
from typing import Optional

import boto3, botocore
from dotenv import load_dotenv

load_dotenv()

def get_bedrock_client(
    runtime: Optional[bool] = True,
):
    region = os.environ.get("AWS_BEDROCK_REGION")
    ak = os.environ.get("AWS_BEDROCK_ACCESS_KEY")
    sk = os.environ.get("AWS_BEDROCK_SECRET_KEY")
    role_arn = os.environ.get("AWS_BEDROCK_ROLE_ARN")
    role_region = os.environ.get("AWS_BEDROCK_ROLE_REGION")
    if runtime:
        service_name='bedrock-runtime'
    else:
        service_name='bedrock'

    try:
        # 1. 首先创建STS client用于获取临时凭证
        sts_client = boto3.client(
            'sts',
            aws_access_key_id=ak,
            aws_secret_access_key=sk,
            region_name=region
        )
        
        # 2. 使用STS assume role获取临时凭证
        assumed_role_object = sts_client.assume_role(
            RoleArn=role_arn,
            RoleSessionName="AssumeRoleSession1"  # 可以自定义session名称
        )
        
        # 3. 从assumed role响应中获取临时凭证
        credentials = assumed_role_object['Credentials']
        
        # 4. 使用临时凭证创建目标服务的client
        role_client = boto3.client(
            service_name,
            aws_access_key_id=credentials['AccessKeyId'],
            aws_secret_access_key=credentials['SecretAccessKey'],
            aws_session_token=credentials['SessionToken'],
            region_name=role_region
        )
        
        return role_client
    except Exception as e:
        print(f"创建client失败: {str(e)}")
        return None

bedrock_runtime = get_bedrock_client()

try:
    response = bedrock_runtime.invoke_model_with_response_stream(
        body=body, modelId=modelId, accept=accept, contentType=contentType
    )
    stream = response.get('body')
    output = []
    
    if stream:
        for event in stream:
            chunk = event.get('chunk')
            if chunk:
                chunk_obj = json.loads(chunk.get('bytes').decode())
                text = chunk_obj['completion']
                print(text)
except botocore.exceptions.ClientError as error:
    if error.response['Error']['Code'] == 'AccessDeniedException':
           print(error.response['Error']['Message'])
        
    else:
        raise error