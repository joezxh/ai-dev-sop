# 第15章：智能体间通信（A2A）

第15 章：智能体间通信（A2A）
单个智能体即使具备强大能力，在面对复杂、多层次问题时仍然存在局限。为了解决这
一难题，Agent 间通信（A2A）使得不同框架构建的智能体能够高效协作，实现无缝协
调、任务委托和信息交换。
Google 的A2A 协议是一项开放标准，旨在促进智能体之间的通用通信。本章将介绍
A2A 的原理、实际应用及其在Google ADK 中的实现。
Agent 间通信模式概述
Agent2Agent（A2A）协议是一项开放标准，旨在实现不同智能体框架之间的通信与协
作。它确保了互操作性，使得基于LangGraph、CrewAI 或Google ADK 等技术开发的
智能体能够跨平台协同工作。
A2A 得到了众多科技公司和服务商的支持，包括Atlassian、Box、LangChain、
MongoDB、Salesforce、SAP 和ServiceNow。微软计划将A2A 集成到Azure AI
Foundry 和Copilot Studio，彰显了对开放协议的重视。此外，Auth0 和SAP 也在其平
台和智能体中集成了A2A 支持。
作为开源协议，A2A 鼓励社区贡献，推动其不断发展和广泛应用。
A2A 核心概念
A2A 协议为智能体交互提供了结构化方法，包含多个核心概念。理解这些基础对于开发
或集成A2A 兼容系统至关重要。A2A 的基础包括核心参与者、Agent Card、Agent 发
现、通信与任务、交互机制和安全性，下面将详细介绍。
核心参与者：A2A 涉及三类主要实体：
・用户：发起智能体协助请求。
・A2A 客户端（客户端Agent）：代表用户请求操作或信息的应用或智能体。
・A2A 服务器（远程Agent）：提供HTTP 端点以处理客户端请求并返回结果的智能体
或系统。远程智能体作为“黑盒”系统，客户端无需了解其内部实现细节。
Agent Card：Agent 的数字身份由Agent Card 定义，通常为JSON 文件。该文件包含

A2A 核心概念
与客户端交互和自动发现所需的关键信息，如智能体身份、端点URL 和版本号，还包
括支持的能力（如流式传输或推送通知）、具体技能、默认输入/输出模式及认证要求。
以下是
WeatherBot 的Agent Card 示例：
WeatherBot Agent Card 示例
# WeatherBot Agent Card 示例
agent_card = {
"name": "WeatherBot",
"description": "提供准确的天气预报和历史数据。",
"url": "http://weather‑service.example.com/a2a",
"version": "1.0.0",
"capabilities": {
"streaming": True,
"pushNotifications": False,
"stateTransitionHistory": True
},
"authentication": {
"schemes": [
"apiKey"
]
},
"defaultInputModes": [
"text"
],
"defaultOutputModes": [
"text"
],
"skills": [
{
"id": "get_current_weather",
"name": "获取当前天气",
"description": "检索任意地点的实时天气。",
"inputModes": [
"text"
],
"outputModes": [
"text"
],
"examples": [
"巴黎现在的天气如何？",
"东京当前天气状况"
],
"tags": [
"weather",
"current",
"real‑time"
]
},
{
"id": "get_forecast",
"name": "获取天气预报",

第15 章：智能体间通信（A2A）
"description": "获取5 天的天气预测。",
"inputModes": [
"text"
],
"outputModes": [
"text"
],
"examples": [
"纽约未来5 天的天气预报",
"伦敦本周末会下雨吗？"
],
"tags": [
"weather",
"forecast",
"prediction"
]
}
]
}
Agent 发现：客户端可通过多种方式发现Agent Card，了解可用A2A 服务器的能力：
・Well‑Known URI：Agent 在标准路径（如
/.well‑known/agent.json ）托管Agent
Card，便于公开或域内自动访问。
・管理型注册表：集中式目录，Agent 可在此发布Agent Card，并按条件查询，适合企
业环境的集中管理与访问控制。
・直接配置：Agent Card 信息嵌入或私下共享，适用于紧密耦合或私有系统，无需动
态发现。
无论采用哪种方式，都应保障Agent Card 端点安全，可通过访问控制、双向TLS
（mTLS）或网络限制实现，尤其当卡片包含敏感（但非密钥）信息时。
通信与任务：在A2A 框架中，通信围绕异步任务展开，任务是长流程的基本工作单元。
每个任务有唯一标识，并经历提交、处理中、完成等状态，支持复杂操作的并行处理。
Agent 间通过消息进行通信。
消息包含属性（如优先级、创建时间等元数据）和一个或多个内容部分（如文本、文件
或结构化JSON 数据）。Agent 在任务中生成的实际输出称为artifact（工件），与消息
类似也由多个部分组成，可按需流式传输。所有A2A 通信均通过HTTP(S) 并采用
JSON‑RPC 2.0 协议。为保持多次交互的上下文，服务器会生成contextId 以关联相关
任务。
交互机制：请求/响应（轮询）、服务器推送事件（SSE）。A2A 提供多种交互方式，满足
不同AI 应用需求：

A2A 核心概念
・同步请求/响应：适用于快速操作，客户端发送请求并等待服务器一次性返回完整
响应。
・异步轮询：适合耗时任务，客户端发送请求，服务器立即返回“处理中”状态和任务
ID，客户端可定期轮询任务状态，直到完成或失败。
・流式更新（SSE）：适用于实时、增量结果，建立服务器到客户端的单向持久连接，
服务器可持续推送状态或部分结果，无需客户端多次请求。
・推送通知（Webhook）：适合超长或资源密集型任务，客户端注册webhook URL，
服务器在任务状态显著变化时异步推送通知。
Agent Card 会声明智能体是否支持流式传输或推送通知。A2A 支持多模态数据（如文
本、音频、视频），可实现丰富的AI 应用。
同步请求示例
# 同步请求示例
sync_request = {
"jsonrpc": "2.0",
"id": "1",
"method": "sendTask",
"params": {
"id": "task‑001",
"sessionId": "session‑001",
"message": {
"role": "user",
"parts": [
{
"type": "text",
"text": "美元兑欧元汇率是多少？"
}
]
},
"acceptedOutputModes": ["text/plain"],
"historyLength": 5
}
}
同步请求使用
sendTask 方法，客户端期望一次性获得完整答案。流式请求则用
sendTaskSubscribe 方法建立持久连接，Agent 可持续返回多次增量结果。
流式请求示例

第15 章：智能体间通信（A2A）
# 流式请求示例
streaming_request = {
"jsonrpc": "2.0",
"id": "2",
"method": "sendTaskSubscribe",
"params": {
"id": "task‑002",
"sessionId": "session‑001",
"message": {
"role": "user",
"parts": [
{
"type": "text",
"text": "今天日元兑英镑汇率是多少？"
}
]
},
"acceptedOutputModes": ["text/plain"],
"historyLength": 5
}
}
安全性：Agent 间通信（A2A）是系统架构的重要组成部分，确保智能体间数据安全、
可靠交换，具备多项内置机制：
・双向TLS：建立加密和认证连接，防止未授权访问和数据泄露，保障通信安全。
・完整审计日志：记录所有智能体间通信，包括信息流、参与智能体和操作，便于审
计、排查和安全分析。
・Agent Card 声明：认证要求在Agent Card 中明确声明，集中管理智能体身份、能力
和安全策略。
・凭证处理：Agent 通常通过OAuth 2.0 令牌或API Key 认证，凭证通过HTTP 头传
递，避免暴露在URL 或消息体中，提高安全性。
A2A 与MCP 对比
A2A 协议与Anthropic 的Model Context Protocol（MCP）互为补充（见图1）。MCP 关
注智能体与外部数据和工具的上下文结构化，而A2A 专注于智能体间的协调与通信，
实现任务委托与协作。
A2A 的目标是提升效率、降低集成成本、促进创新和互操作性，助力复杂多智能体系统
开发。因此，深入理解A2A 的核心组件和运行方式，是设计、实现和应用协作型智能体
系统的基础。

实践应用与场景
图1：A2A 与MCP 协议对比
实践应用与场景
Agent 间通信是构建复杂AI 解决方案不可或缺的基础，带来模块化、可扩展性和智能
增强。
・多框架协作：A2A 的核心应用是让不同框架（如ADK、LangChain、CrewAI）构建的
独立智能体实现通信与协作。适用于多智能体系统，各智能体专注于问题的不同
方面。
・自动化工作流编排：在企业场景下，A2A 可实现智能体间任务委托与协调。例如，一
个智能体负责数据采集，另一个负责分析，第三个生成报告，三者通过A2A 协议协
同完成复杂流程。
・动态信息检索：Agent 可互相请求和交换实时信息。主智能体可向专门的数据获取智
能体请求市场数据，后者通过外部API 获取并返回结果。

第15 章：智能体间通信（A2A）
实战代码示例
A2A 协议的实际应用可参考samples，其中包含Java、Go 和Python 示例，展示
LangGraph、CrewAI、Azure AI Foundry、AG2 等框架智能体如何通过A2A 通信。所有
代码均采用Apache 2.0 许可。以下以ADK 智能体为例，介绍如何用Google 认证工具
搭建A2A 服务器。完整代码见GitHub。
ADK 智能体创建示例
import datetime
from google.adk.agents import LlmAgent # type: ignore[import‑untyped]
from google.adk.tools.google_api_tool import CalendarToolset # type: ignore[import‑untyped]
async def create_agent(client_id, client_secret) ‑> LlmAgent:
"""构建ADK Agent。"""
toolset = CalendarToolset(client_id=client_id, client_secret=client_secret)
return LlmAgent(
model='gemini‑2.0‑flash‑001',
name='calendar_agent',
description="可帮助管理用户日历的Agent",
instruction=f"""
13 你是一个可以帮助用户管理日历的Agent。
15 用户会请求日历状态信息或修改日历。请使用提供的工具与日历API 交互。
17 如未指定，默认使用'primary' 日历。
19 使用Calendar API 工具时，请采用规范的RFC3339 时间戳。
21 今天是{datetime.datetime.now()}。
""",
tools=await toolset.get_tools(),
)
上述Python 代码定义了异步函数
create_agent ，用于构建ADK
LlmAgent 。首先通
过客户端凭证初始化
CalendarToolset ，访问Google Calendar API。随后创建
LlmAgent 实例，配置Gemini 模型、名称和管理日历的说明，并集成
CalendarToolset 工具，实现日历查询和修改。说明中动态插入当前日期，便于时序
上下文。
以下代码展示了如何定义智能体的具体说明和工具，完整文件见GitHub。
A2A 服务器主函数示例

实战代码示例
import os
import asyncio
import uvicorn
from starlette.applications import Starlette
from starlette.routing import Route
from starlette.requests import Request
from starlette.responses import PlainTextResponse
# 假设这些导入来自相关的A2A 和ADK 库
from a2a import (AgentSkill, AgentCard, AgentCapabilities,
DefaultRequestHandler, A2AStarletteApplication)
from adk import (Runner, InMemoryArtifactService, InMemorySessionService,
InMemoryMemoryService, ADKAgentExecutor, InMemoryTaskStore)
def main(host: str, port: int):
# 检查API Key 是否设置。
# 使用Vertex AI API 时无需设置。
if os.getenv('GOOGLE_GENAI_USE_VERTEXAI') != 'TRUE' and not os.getenv(
'GOOGLE_API_KEY'
):
raise ValueError(
' 未设置GOOGLE_API_KEY 环境变量，且GOOGLE_GENAI_USE_VERTEXAI 不是TRUE。'
)
skill = AgentSkill(
id='check_availability',
name=' 检查可用性',
description="使用Google Calendar 检查用户某一时间段的空闲情况",
tags=['calendar'],
examples=[' 我明天上午10 点到11 点有空吗？'],
)
agent_card = AgentCard(
name='Calendar Agent',
description="可管理用户日历的Agent",
url=f'http://{host}:{port}/',
version='1.0.0',
defaultInputModes=['text'],
defaultOutputModes=['text'],
capabilities=AgentCapabilities(streaming=True),
skills=[skill],
)
adk_agent = asyncio.run(create_agent(
client_id=os.getenv('GOOGLE_CLIENT_ID'),
client_secret=os.getenv('GOOGLE_CLIENT_SECRET'),
))
runner = Runner(
app_name=agent_card.name,
agent=adk_agent,
artifact_service=InMemoryArtifactService(),
session_service=InMemorySessionService(),
memory_service=InMemoryMemoryService(),
)
agent_executor = ADKAgentExecutor(runner, agent_card)

第15 章：智能体间通信（A2A）
async def handle_auth(request: Request) ‑> PlainTextResponse:
await agent_executor.on_auth_callback(
str(request.query_params.get('state')), str(request.url)
)
return PlainTextResponse(' 认证成功。')
request_handler = DefaultRequestHandler(
agent_executor=agent_executor, task_store=InMemoryTaskStore()
)
a2a_app = A2AStarletteApplication(
agent_card=agent_card, http_handler=request_handler
)
routes = a2a_app.routes()
routes.append(
Route(
path='/authenticate',
methods=['GET'],
endpoint=handle_auth,
)
)
app = Starlette(routes=routes)
uvicorn.run(app, host=host, port=port)
if __name__ == '__main__':
main()
上述Python 代码演示了如何搭建一个符合A2A 协议的“日历Agent”，用于检查用户
日历空闲时间。包括API Key 或Vertex AI 配置认证、AgentCard 能力和技能定义、
ADK 智能体创建、内存服务配置、Starlette Web 应用初始化、认证回调和A2A 协议处
理，并通过Uvicorn 以HTTP 方式暴露智能体服务。
这些示例展示了从能力定义到Web 服务运行的A2A 智能体构建流程。通过智能体Card
和ADK，开发者可创建可与Google Calendar 等工具集成的互操作智能体，构建多智
能体生态系统。
更多A2A 实践可参考How to Build Your First Google A2A Project: A Step‑by‑Step
Tutorial，该链接提供Python 和JavaScript 示例客户端与服务器、多智能体Web 应
用、命令行工具及多种框架实现。
一图速览
是什么：不同框架构建的单一智能体在面对复杂、多层次问题时常常力不从心。主要挑
战在于缺乏统一协议，无法高效沟通与协作，导致各自为政，难以组合专长解决更大任

关键要点
务。没有标准化方法，集成成本高、周期长，阻碍了更强大、协同AI 系统的开发。
为什么：Agent 间通信（A2A）协议为此问题提供了开放、标准化解决方案。它基于
HTTP 协议，实现互操作性，使不同技术栈的智能体能够无缝协调、任务委托和信息共
享。核心组件是Agent Card，描述智能体能力、技能和通信端点，便于发现和交互。
A2A 支持同步和异步多种交互机制，满足多样化场景。统一标准促进了模块化、可扩展
的多智能体系统生态。
使用原则：当需要编排两个或以上智能体协作，尤其是跨框架（如Google ADK、
LangGraph、CrewAI）时，建议采用此模式。适合构建复杂、模块化应用，各智能体专
注于工作流不同环节，如数据分析委托给一个智能体，报告生成交由另一个。Agent 需
动态发现和调用其他智能体能力时也适用。
视觉摘要
图2：A2A 智能体间通信模式
关键要点
・Google A2A 协议是一项开放、基于HTTP 的标准，促进不同框架智能体间的通信与
协作。
・AgentCard 是智能体的数字身份，便于其他智能体自动发现和理解其能力。

第15 章：智能体间通信（A2A）
・A2A 支持同步请求‑ 响应（
tasks/send ）和流式更新（
tasks/sendSubscribe ），
满足不同通信需求。
・协议支持多轮对话，包括
input‑required 状态，Agent 可请求补充信息并保持上
下文。
・A2A 鼓励模块化架构，专用智能体可独立运行于不同端口，实现系统可扩展和分布
式部署。
・Trickle AI 等工具可可视化和跟踪A2A 通信，便于开发者监控、调试和优化多智能体
系统。
・A2A 专注于智能体间任务和工作流管理，MCP 则为LLM 与外部资源交互提供标准
接口。
总结
Agent 间通信（A2A）协议为打破单体智能体孤岛提供了关键开放标准。通过统一的
HTTP 框架，实现了不同平台（如Google ADK、LangGraph、CrewAI）Agent 的无缝协
作与互操作。Agent Card 作为数字身份，清晰定义智能体能力，支持动态发现。协议灵
活，涵盖同步请求、异步轮询和实时流式等多种交互模式，满足广泛应用需求。
A2A 支持模块化、可扩展架构，专用智能体可组合编排复杂自动化流程。安全性为核
心，内置mTLS 和认证机制保障通信安全。A2A 与MCP 等标准互补，专注于高层智能
体协调与任务委托。主流科技公司支持和丰富实践案例，彰显其重要性。A2A 为开发者
构建更复杂、分布式、智能化多智能体系统奠定了基础，是协作型AI 生态的关键支柱。
参考文献
・陈博（2025 年4 月22 日）《Google A2A 项目入门教程》‑ trickle.so
・Google A2A GitHub 仓库‑ github.com
・Google Agent Development Kit (ADK) ‑ google.github.io
・Agent‑to‑Agent (A2A) 协议入门‑ codelabs.developers.google.com
・Google AgentDiscovery ‑ a2a‑protocol.org
・LangGraph、CrewAI、Google ADK 等框架智能体间通信‑ trickle.so
・使用A2A 协议设计协作型多智能体系统‑ oreilly.com

第16 章：资源感知优化
资源感知优化使智能体能够在运行过程中动态监控和管理计算、时间和财务资源。这与
仅关注动作序列的简单规划不同，资源感知优化要求智能体在执行动作时做出决策，以
在指定资源预算内实现目标或优化效率。这包括在更准确但昂贵的模型与更快、低成本
模型之间进行选择，或决定是否分配更多算力以获得更精细的响应，还是返回更快但较
为粗略的答案。
例如，假设一个智能体为金融分析师分析大型数据集。如果分析师需要立即获得初步报
告，智能体可能会使用更快、更经济的模型快速总结关键趋势。而如果分析师需要用于
重要投资决策的高精度预测，并且有更充足的预算和时间，智能体则会分配更多资源，
采用功能更强大但速度较慢的高精度预测模型。此类场景中的关键策略是回退机制：当
首选模型因过载或限流不可用时，系统自动切换到默认或更经济的模型，保证服务连续
性而非完全失败。
实践应用与用例
实际应用场景包括：
・成本优化的LLM 使用：智能体根据预算约束，决定复杂任务使用大型昂贵LLM，简
单查询则用小型经济模型。
・延迟敏感操作：在实时系统中，智能体选择更快但可能不够全面的推理路径，以确
保及时响应。
・能效优化：部署在边缘设备或电量有限环境下的智能体，通过优化处理流程节省电
池寿命。
・服务可靠性回退：当主模型不可用时，智能体自动切换到备选模型，确保服务不中
断并实现优雅降级。
・数据使用管理：智能体选择摘要数据而非完整数据集下载，以节省带宽或存储空间。
・自适应任务分配：在多智能体系统中，智能体根据自身算力负载或可用时间自我分
配任务。