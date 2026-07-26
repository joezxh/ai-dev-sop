# 第8章：记忆管理

第8 章：记忆管理
高效的记忆管理对于智能体保留信息至关重要。智能体需要不同类型的记忆，就像人类
一样，以实现高效运作。本章将深入探讨记忆管理，重点介绍智能体的即时（短期）和
持久（长期）记忆需求。
在智能体系统中，记忆指的是智能体保留并利用过去交互、观察和学习经验的信息能
力。这使智能体能够做出明智决策、保持对话上下文，并不断提升自身能力。智能体的
记忆通常分为两大类：
・短期记忆（上下文记忆）：类似于工作记忆，保存当前正在处理或最近访问的信息。
对于使用大语言模型（LLM）的智能体来说，短期记忆主要体现在上下文窗口中。该
窗口包含最近的消息、智能体回复、工具使用结果以及当前交互中的智能体反思，
这些内容共同影响LLM 的后续响应和行为。上下文窗口容量有限，限制了智能体可
直接访问的最近信息量。高效的短期记忆管理需要在有限空间内保留最相关的信息，
常用方法包括对旧对话片段进行摘要或突出关键信息。具备“长上下文”窗口的新
模型仅仅扩大了短期记忆的容量，使单次交互可容纳更多信息，但这些上下文依然
是临时的，会在会话结束后丢失，并且每次处理全部内容成本较高且效率不高。因
此，智能体还需要其他类型的记忆来实现真正的持久性、跨会话的信息回溯，以及
知识库的构建。
・长期记忆（持久记忆）：作为智能体需要在多次交互、任务或较长周期内保留信息的
仓库，类似于长期知识库。数据通常存储在智能体的外部环境中，如数据库、知识图
谱或向量数据库。在向量数据库中，信息被转换为数值向量并存储，智能体可通过
语义相似度而非关键词精确匹配进行检索，这一过程称为语义搜索。当智能体需要
长期记忆中的信息时，会查询外部存储，检索相关数据，并将其整合到短期上下文
中，实现知识的融合。
实践应用与场景
记忆管理对于智能体跟踪信息和实现智能行为至关重要，是智能体超越基础问答能力的
关键。典型应用包括：
・聊天机器人与对话式AI：保持对话连贯性依赖短期记忆，需记住用户先前输入以生
成合理回复。长期记忆则让机器人能回忆用户偏好、历史问题或过往讨论，实现个

第8 章：记忆管理
性化和持续交互。
・任务型智能体：管理多步骤任务时，需用短期记忆跟踪前序步骤、当前进度和总体
目标，这些信息通常存于任务上下文或临时存储。长期记忆则用于访问不在当前上
下文中的用户相关数据。
・个性化体验：提供定制化交互的智能体会利用长期记忆存储和检索用户偏好、历史
行为和个人信息，从而调整响应和建议。
・学习与提升：智能体可通过学习过往交互不断优化表现，将成功策略、错误和新知
识存入长期记忆，便于未来适应。强化学习智能体会以此方式保存学习成果。
・信息检索（RAG）：面向问答的智能体会访问知识库（长期记忆），通常通过检索增强
生成（RAG）实现，智能体检索相关文档或数据以辅助回答。
・自主系统：机器人或自动驾驶汽车需记忆地图、路线、物体位置和学习行为，短期记
忆用于处理即时环境，长期记忆则保存通用环境知识。
记忆让智能体能够维护历史、学习、个性化交互，并处理复杂的时序问题。
实战代码：Google Agent Developer Kit (ADK) 的记忆管理
Google Agent Developer Kit (ADK) 提供了结构化的上下文与记忆管理方法，包括实际
应用组件。理解ADK 的
Session 、State 和
Memory 对于构建需要保留信息的智能
体至关重要。
如同人类交流，智能体需要回忆先前对话以实现连贯自然的交互。ADK 通过三个核心
概念及其服务简化了上下文管理：
每次与智能体的交互都可视为独立的对话线程，智能体可能需要访问早期交互的数据。
ADK 的结构如下：
・Session（会话）：单个聊天线程，记录该次交互的消息和行为（Event），并存储与
该对话相关的临时数据（State）。
・State（session.state）：存储于Session 内，仅与当前活跃聊天线程相关的数据。
・Memory（记忆）：可检索的信息仓库，来源于历史聊天或外部数据，用于超越当前
对话的数据检索。
ADK 提供专用服务管理这些关键组件，便于构建复杂、有状态且具备上下文感知能力的
智能体。SessionService 管理聊天线程（
Session 对象）的创建、记录和终止，
MemoryService 负责长期知识（Memory）的存储与检索。

Session：跟踪每次聊天
SessionService 和
MemoryService 均支持多种存储方式，可根据应用需求选择。内
存存储适合测试，数据不会跨重启保留。生产环境则可选数据库或云服务实现持久化和
可扩展性。
Session：跟踪每次聊天
ADK 的
Session 对象用于跟踪和管理单个聊天线程。每次与智能体开启对话时，
SessionService 会生成一个
Session 对象（
google.adk.sessions.Session ），包
含会话唯一标识（
id 、app_name 、user_id ）、事件记录（
Event ）、会话临时数据
（
state ）及最后更新时间（
last_update_time ）。开发者通常通过
SessionService 间接操作
Session 。SessionService 负责会话生命周期管理，包
括新建、恢复、记录活动（含状态更新）、识别活跃会话及数据清理。ADK 提供多种
SessionService 实现，支持不同的会话历史和临时数据存储方式，如
InMemorySessionService 适合测试但不持久化数据。
# 示例：使用InMemorySessionService
from google.adk.sessions import InMemorySessionService
session_service = InMemorySessionService()
如需可靠持久化，可使用
DatabaseSessionService ：
# 示例：使用DatabaseSessionService
from google.adk.sessions import DatabaseSessionService
db_url = "sqlite:///./my_agent_data.db"
session_service = DatabaseSessionService(db_url=db_url)
此外，VertexAiSessionService 可利用Vertex AI 基础设施实现云端可扩展生产部署。
# 示例：使用VertexAiSessionService
from google.adk.sessions import VertexAiSessionService
PROJECT_ID = "your‑gcp‑project‑id"
LOCATION = "us‑central1"
REASONING_ENGINE_APP_NAME =
"projects/your‑gcp‑project‑id/locations/us‑central1/reasoningEngines/your‑engine‑id"
,!
session_service = VertexAiSessionService(project=PROJECT_ID, location=LOCATION)
# 使用时需传入REASONING_ENGINE_APP_NAME

第8 章：记忆管理
选择合适的
SessionService 决定了智能体交互历史和临时数据的存储方式及持久性。
每次消息交换都涉及循环流程：收到消息后，Runner 通过
SessionService 获取或创
建
Session ，智能体利用
Session 的上下文（状态和历史）处理消息，生成响应并可
能更新状态，Runner 将其封装为
Event ，通过
session_service.append_event 记
录事件并更新状态，Session 等待下一条消息。会话结束时可调用
delete_session
终止会话。此流程确保
SessionService 通过管理会话历史和临时数据实现连续性。
State：会话的临时记事本
在ADK 中，每个
Session （聊天线程）都包含一个
state 组件，类似于智能体在当
前对话期间的临时工作记忆。session.events 记录完整聊天历史，session.state 则
存储和更新与当前会话相关的动态数据。
本质上，session.state 是一个字典，以键值对形式存储数据。其核心作用是让智能体
保留和管理对话所需的细节，如用户偏好、任务进度、数据收集或影响后续行为的
标志。
state 的结构为字符串键和可序列化的Python 基本类型（字符串、数字、布尔值、列
表、字典）。state 是动态的，会话过程中不断变化，持久性取决于所选
SessionService 。
可通过键前缀组织数据范围和持久性。无前缀的键为会话专属：
・
user : 前缀关联用户ID，跨会话共享。
・
app : 前缀为应用级数据，所有用户共享。
・
temp : 前缀为仅本轮处理有效的临时数据，不持久化。
Agent 通过
session.state 字典访问所有状态数据，SessionService 负责数据检索、
合并和持久化。应在通过
session_service.append_event() 添加事件时更新state，
确保准确跟踪、正确保存并安全处理状态变更。
1. 简单方式：使用output_key（用于文本回复）：若仅需将智能体最终文本回复保存
到
state ，可在
LlmAgent 设置
output_key ，Runner 会自动在添加事件时保
存响应到
state 。示例代码如下：
Google ADK 使用output_key 管理状态示例

State：会话的临时记事本
from google.adk.agents import LlmAgent
from google.adk.sessions import InMemorySessionService, Session
from google.adk.runners import Runner
from google.genai.types import Content, Part
greeting_agent = LlmAgent(
name="Greeter",
model="gemini‑2.0‑flash",
instruction="生成简短友好的问候语。",
output_key="last_greeting"
)
app_name, user_id, session_id = "state_app", "user1", "session1"
session_service = InMemorySessionService()
runner = Runner(
agent=greeting_agent,
app_name=app_name,
session_service=session_service
)
session = session_service.create_session(
app_name=app_name,
user_id=user_id,
session_id=session_id
)
print(f"初始state: {session.state}")
user_message = Content(parts=[Part(text="你好")])
print("\n‑‑‑ 运行Agent ‑‑‑")
for event in runner.run(
user_id=user_id,
session_id=session_id,
new_message=user_message
):
if event.is_final_response():
print("Agent 已回复。")
updated_session = session_service.get_session(app_name, user_id, session_id)
print(f"\nAgent 运行后state: {updated_session.state}")
2. 标准方式：使用
EventActions.state_delta （复杂更新）：若需一次更新多个键、
保存非文本内容、指定作用域（如
user: 或
app: ）、或进行与最终回复无关的
更新，可手动构建state_delta 字典并包含在事件的
EventActions 中。示例：
Google ADK 使用state_delta 管理状态示例
import time
from google.adk.tools.tool_context import ToolContext
from google.adk.sessions import InMemorySessionService

第8 章：记忆管理
def log_user_login(tool_context: ToolContext) ‑> dict:
state = tool_context.state
login_count = state.get("user:login_count", 0) + 1
state["user:login_count"] = login_count
state["task_status"] = "active"
state["user:last_login_ts"] = time.time()
state["temp:validation_needed"] = True
print("已在工具内更新state。")
return {
"status": "success",
"message": f"用户登录已记录，总次数：{login_count}。"
}
session_service = InMemorySessionService()
app_name, user_id, session_id = "state_app_tool", "user3", "session3"
session = session_service.create_session(
app_name=app_name,
user_id=user_id,
session_id=session_id,
state={"user:login_count": 0, "task_status": "idle"}
)
print(f"初始state: {session.state}")
from google.adk.tools.tool_context import InvocationContext
mock_context = ToolContext(
invocation_context=InvocationContext(
app_name=app_name, user_id=user_id, session_id=session_id,
session=session, session_service=session_service
)
)
log_user_login(mock_context)
updated_session = session_service.get_session(app_name, user_id, session_id)
print(f"工具执行后state: {updated_session.state}")
此代码展示了通过工具封装状态变更的推荐做法。函数
log_user_login 作为工具，
负责在用户登录时更新
session.state ，包括登录次数、任务状态、最后登录时间和
临时标志。演示部分模拟了工具的实际调用流程，最终展示
state 已被工具更新。
请注意，直接修改
session.state 字典（如
session = session_service.get_session(...) 后直接赋值）是不推荐的，因为这样
会绕过标准事件处理机制，导致变更未被记录、无法持久化、可能引发并发问题且不会
更新元数据。推荐的状态更新方式是通过
output_key 或在
append_event 时包含
state_delta 。session.state 应主要用于读取数据。
设计
state 时应保持简单，使用基础类型、清晰命名和正确前缀，避免深层嵌套，并
始终通过
append_event 流程更新。

Memory：MemoryService 管理长期知识
Memory：MemoryService 管理长期知识
在智能体系统中，Session 组件维护当前聊天历史（events）和临时数据（state），但
要跨多次交互或访问外部数据，则需长期知识管理，由MemoryService 实现。
# 示例：使用InMemoryMemoryService
from google.adk.memory import InMemoryMemoryService
memory_service = InMemoryMemoryService()
Session 和
State 可视为单次会话的短期记忆，而
MemoryService 管理的长期知
识则是持久且可检索的信息仓库，可能包含多次交互或外部数据。MemoryService
（
BaseMemoryService 接口）定义了长期知识管理标准，主要功能包括添加信息
（
add_session_to_memory ）和检索信息（
search_memory ）。
ADK 提供多种长期知识存储实现，InMemoryMemoryService 适合测试但不持久化。生
产环境推荐使用
VertexAiRagMemoryService ，利用Google Cloud 的RAG 服务实现可
扩展、持久和语义检索（详见第十四章RAG）。
# 示例：使用VertexAiRagMemoryService
from google.adk.memory import VertexAiRagMemoryService
RAG_CORPUS_RESOURCE_NAME =
"projects/your‑gcp‑project‑id/locations/us‑central1/ragCorpora/your‑corpus‑id"
,!
SIMILARITY_TOP_K = 5
VECTOR_DISTANCE_THRESHOLD = 0.7
memory_service = VertexAiRagMemoryService(
rag_corpus=RAG_CORPUS_RESOURCE_NAME,
similarity_top_k=SIMILARITY_TOP_K,
vector_distance_threshold=VECTOR_DISTANCE_THRESHOLD
)
实战代码：LangChain 与LangGraph 的记忆管理
在LangChain 和LangGraph 中，记忆是构建智能、自然对话应用的关键。它让智能体
能记住过往交互、学习反馈并适应用户偏好。LangChain 的记忆功能通过引用历史丰富
当前提示，并记录最新交互以备后用。随着任务复杂度提升，这一能力对效率和用户体
验至关重要。
短期记忆：线程级别，跟踪单次会话内的对话。为LLM 提供即时上下文，但完整历史

第8 章：记忆管理
可能超出上下文窗口，导致错误或性能下降。LangGraph 将短期记忆作为智能体状态
的一部分，通过
checkpointer 持久化，可随时恢复线程。
长期记忆：跨会话保存用户或应用级数据，在自定义命名空间下存储，可随时在任意线
程中调用。LangGraph 提供存储工具，支持长期记忆的保存与检索，实现知识的永久
保留。
LangChain 提供多种对话历史管理工具，从手动到自动集成于链中。
ChatMessageHistory：手动管理对话历史。适合在链外直接控制对话历史。
from langchain.memory import ChatMessageHistory
history = ChatMessageHistory()
history.add_user_message("我下周要去纽约。")
history.add_ai_message("太棒了！纽约是个很棒的城市。")
print(history.messages)
ConversationBuerMemory：链式自动记忆。集成于链中，自动保存对话历史并注入
到提示中。主要参数：
・
memory_key ：指定提示中保存历史的变量名，默认
history 。
・
return_messages ：布尔值，决定历史格式。False 返回格式化字符串，适合标准
LLM；True 返回消息对象列表，推荐用于Chat Model。
from langchain.memory import ConversationBufferMemory
memory = ConversationBufferMemory()
memory.save_context({"input": "天气怎么样？"}, {"output": "今天晴天。"})
print(memory.load_memory_variables({}))
集成到LLMChain，可让模型访问历史并生成有上下文的回复：
LangChain ConversationBuerMemory 示例
from langchain_openai import OpenAI
from langchain.chains import LLMChain
from langchain.prompts import PromptTemplate
from langchain.memory import ConversationBufferMemory
llm = OpenAI(temperature=0)
template = """你是一名乐于助人的旅行顾问。

实战代码：LangChain 与LangGraph 的记忆管理
9 之前的对话：
{history}
12 新问题：{question}
13 回复："""
prompt = PromptTemplate.from_template(template)
memory = ConversationBufferMemory(memory_key="history")
conversation = LLMChain(llm=llm, prompt=prompt, memory=memory)
response = conversation.predict(question="我想订机票。")
print(response)
response = conversation.predict(question="顺便说一下，我叫Sam。")
print(response)
response = conversation.predict(question="你还记得我的名字吗？")
print(response)
对于Chat Model，建议设置
return_messages=True ，以结构化消息对象提升效果。
LangChain Chat Model 记忆管理示例
from langchain_openai import ChatOpenAI
from langchain.chains import LLMChain
from langchain.memory import ConversationBufferMemory
from langchain_core.prompts import (
ChatPromptTemplate,
MessagesPlaceholder,
SystemMessagePromptTemplate,
HumanMessagePromptTemplate,
)
llm = ChatOpenAI()
prompt = ChatPromptTemplate(
messages=[
SystemMessagePromptTemplate.from_template("你是一名友好的助手。"),
MessagesPlaceholder(variable_name="chat_history"),
HumanMessagePromptTemplate.from_template("{question}")
]
)
memory = ConversationBufferMemory(memory_key="chat_history", return_messages=True)
conversation = LLMChain(llm=llm, prompt=prompt, memory=memory)
response = conversation.predict(question="你好，我是Jane。")
print(response)
response = conversation.predict(question="你还记得我的名字吗？")
print(response)
长期记忆类型：长期记忆让系统能跨会话保留信息，实现更深层次的上下文和个性化。
主要分为三类，类比人类记忆：
・语义记忆：记住事实。保存具体事实和概念，如用户偏好或领域知识，用于提升智能

第8 章：记忆管理
体回复的个性化和相关性。可管理为用户“画像”（JSON 文档）或事实集合。
・情景记忆：记住经历。回忆过去事件或行为，常用于智能体记住任务完成方式。实际
应用中常通过few‑shot 示例提示实现，让智能体学习成功交互序列。
・程序性记忆：记住规则。记忆任务执行方法，即智能体的核心指令和行为，通常存于
系统提示。智能体可通过“反思”技术自我优化指令。
以下伪代码演示智能体如何通过反思更新程序性记忆，并存储于LangGraph
BaseStore ：
LangGraph 反思更新程序性记忆示例
from typing import TypedDict
from langgraph.store.base import BaseStore
class State(TypedDict):
messages: list
def update_instructions(state: State, store: BaseStore, prompt_template, llm):
namespace = ("instructions",)
current_instructions = store.search(namespace)[0]
prompt = prompt_template.format(
instructions=current_instructions.value["instructions"],
conversation=state["messages"]
)
output = llm.invoke(prompt)
new_instructions = output['new_instructions']
store.put(("agent_instructions",), "agent_a", {"instructions": new_instructions})
def call_model(state: State, store: BaseStore, prompt_template):
namespace = ("agent_instructions", )
instructions = store.get(namespace, key="agent_a")[0]
prompt = prompt_template.format(instructions=instructions.value["instructions"])
# ... 业务逻辑继续
LangGraph 将长期记忆以JSON 文档存储，每条记忆按命名空间（如文件夹）和键（如
文件名）组织，便于检索。以下代码演示InMemoryStore 的put、get 和search 用法：
LangGraph InMemoryStore 长期记忆示例
from langgraph.store.memory import InMemoryStore
def embed(texts: list[str]) ‑> list[list[float]]:
return [[1.0, 2.0] for _ in texts]
store = InMemoryStore(index={"embed": embed, "dims": 2})
user_id = "my‑user"

Vertex Memory Bank
application_context = "chitchat"
namespace = (user_id, application_context)
store.put(
namespace,
"a‑memory",
{
"rules": [
"用户喜欢简短直接的语言",
"用户只说英语和python",
],
"my‑key": "my‑value",
},
)
item = store.get(namespace, "a‑memory")
print("检索结果：", item)
items = store.search(
namespace,
filter={"my‑key": "my‑value"},
query="语言偏好"
)
print("搜索结果：", items)
Vertex Memory Bank
Memory Bank 是Vertex AI Agent Engine 的托管服务，为智能体提供持久的长期记忆。
服务利用Gemini 模型异步分析对话历史，提取关键事实和用户偏好。
信息按作用域（如用户ID）持久存储，并智能更新以整合新数据和解决矛盾。新会话开
始时，智能体可通过全量回调或嵌入相似度检索相关记忆，实现跨会话连续性和个性化
响应。
Agent 的runner 通过
VertexAiMemoryBankService 交互，初始化后自动存储会话生成
的记忆，每条记忆标记唯一
USER_ID 和
APP_NAME ，确保未来准确检索。
Vertex AI Memory Bank 集成示例
import asyncio
from google.adk.memory import VertexAiMemoryBankService
async def setup_memory_bank(agent_engine, session_service, app_name, session):
agent_engine_id = agent_engine.api_resource.name.split("/")[‑1]
memory_service = VertexAiMemoryBankService(
project="PROJECT_ID",
location="LOCATION",
agent_engine_id=agent_engine_id
)

第8 章：记忆管理
session = await session_service.get_session(
app_name=app_name,
user_id="USER_ID",
session_id=session.id
)
await memory_service.add_session_to_memory(session)
Memory Bank 与Google ADK 无缝集成，开箱即用。对于LangGraph、CrewAI 等其他
智能体框架，也可通过API 直接调用，相关代码示例可在线查阅。
一图速览
是什么：智能体系统需要记住过往交互信息，才能完成复杂任务并提供连贯体验。没有
记忆机制，智能体就是无状态的，无法保持对话上下文、学习经验或个性化响应，仅能
处理简单的一问一答，无法应对多步骤流程或用户需求变化。核心问题是如何高效管理
单次会话的临时信息和长期积累的知识。
为什么：标准解决方案是实现双组件记忆系统，区分短期与长期存储。短期记忆保存最
近交互数据于LLM 上下文窗口，维持对话流畅。需持久的信息则通过外部数据库（常
为向量库）实现高效语义检索。ADK 等智能体框架提供专用组件管理记忆，如
Session （会话线程）、State （临时数据）和
MemoryService （长期知识库接口），
智能体可检索并融合历史信息到当前上下文。
经验法则：只要智能体需要做的不仅仅是回答单个问题，就应采用该模式。对于需在对
话中保持上下文、跟踪多步骤任务进度或通过回忆用户偏好和历史实现个性化的
Agent，记忆管理是必需的。只要智能体需根据过往成功、失败或新知识学习和适应，
都应实现记忆管理。
视觉总结
关键要点
快速回顾记忆管理的核心内容：
・记忆对于智能体跟踪信息、学习和个性化交互至关重要。
・对话式AI 依赖短期记忆维持单次聊天上下文，长期记忆则跨会话保存知识。
・短期记忆（即时信息）是临时的，常受LLM 上下文窗口或框架传递方式限制。
・长期记忆（持久信息）通过外部存储（如向量数据库）保存，并通过检索访问。

总结
图1：记忆管理设计模式
・ADK 框架有专用组件：Session（聊天线程）、State（临时数据）、MemoryService
（长期知识库）管理记忆。
・ADK 的
SessionService 管理会话生命周期，包括历史（
events ）和临时数据
（
state ）。
・ADK 的
session.state 是临时数据字典，前缀（
user: 、app: 、temp: ）标明数
据归属和持久性。
・在ADK 中，推荐通过
EventActions.state_delta 或
output_key 更新
state ，
不要直接修改state 字典。
・ADK 的
MemoryService 用于长期存储和检索信息，常通过工具实现。
・LangChain 提供如
ConversationBufferMemory 等工具，自动将单次对话历史注入
提示，实现即时上下文回忆。
・LangGraph 支持高级长期记忆，通过store 保存和检索语义事实、情景经历或可更
新规则，跨用户会话持久化。
・Memory Bank 是托管服务，自动提取、存储和回忆用户信息，实现个性化、持续对
话，支持ADK、LangGraph、CrewAI 等框架。
总结
本章深入讲解了智能体系统的记忆管理，阐明了短期上下文与长期知识的区别及其实现
方式。介绍了Google ADK 提供的Session、State 和MemoryService 等组件的具体用

第8 章：记忆管理
法。掌握了智能体如何记住信息后，下一章将进入“学习与适应”模式，探讨智能体如
何根据新经验或数据改变思维、行为和知识。
参考文献
・ADK Memory – Google ADK 文档
・LangGraph Memory – LangGraph 概念
・Vertex AI Agent Engine Memory Bank – Google Cloud 博客