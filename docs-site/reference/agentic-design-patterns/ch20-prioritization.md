# 第20章：优先级排序

第20 章：优先级排序
在复杂且动态的环境中，智能体常常面临大量潜在行动、目标冲突和资源有限的问题。
如果没有明确的后续行动决策流程，智能体可能会效率低下、操作延迟，甚至无法实现
关键目标。优先级排序模式通过让智能体根据任务的重要性、紧急性、依赖关系和既定
标准进行评估和排序，解决了这一问题。这样可以确保智能体将精力集中在最关键的任
务上，从而提升整体效能和目标达成度。
优先级排序模式概述
智能体通过优先级排序，有效管理任务、目标和子目标，引导后续行动。在面对多重需
求时，这一过程帮助智能体做出明智决策，将重要或紧急事项优先处理，次要任务则延
后。该模式尤其适用于资源有限、时间紧迫、目标可能冲突的真实场景。
智能体优先级排序的核心通常包括几个要素。首先，标准定义用于建立任务评估的规则
或指标，如紧急性（任务的时间敏感度）、重要性（对主要目标的影响）、依赖关系（是
否为其他任务的前置条件）、资源可用性（所需工具或信息的准备情况）、成本/收益分
析（投入与预期结果）、以及个性化智能体的用户偏好。其次，任务评估是指根据这些
标准对每个潜在任务进行分析，方法可以从简单规则到复杂的评分体系或LLM 推理。
第三，调度或选择逻辑是指根据评估结果选择最佳下一步行动或任务顺序，可能采用队
列或高级规划组件。最后，动态优先级调整允许智能体在环境变化时修改任务优先级，
如出现新的关键事件或临近截止时间，确保智能体具备适应性和响应能力。
优先级排序可发生在多个层级：选择总体目标（高层级目标排序）、规划步骤排序（子
任务排序）、或从可选项中选择下一步行动（行动选择）。有效的优先级排序让智能体在
复杂、多目标环境下表现得更智能、高效和稳健。这类似于人类团队管理者会根据成员
意见对任务进行排序。
实践应用与场景
在各种实际应用中，智能体通过优先级排序实现高效、及时的决策：
・自动化客户支持：智能体优先处理紧急请求（如系统故障报告），而将常规问题（如
密码重置）延后。还可对高价值客户给予优先响应。

实战代码示例
・云计算资源调度：AI 在高峰时段优先分配资源给关键应用，将低优先级批处理任务
安排在非高峰期，以优化成本。
・自动驾驶系统：持续优先考虑安全和效率。例如，避免碰撞的制动优先于保持车道
或优化油耗。
・金融交易：智能体根据市场状况、风险容忍度、利润率和实时新闻优先执行高优先
级交易。
・项目管理：智能体根据截止日期、依赖关系、团队可用性和战略重要性对项目任务
进行排序。
・网络安全：智能体监控网络流量时，根据威胁严重性、潜在影响和资产关键性优先
处理警报，确保对最危险威胁及时响应。
・个人助理AI：通过优先级排序管理日常事务，根据用户定义的重要性、临近截止时
间和当前上下文安排日程、提醒和通知。
这些案例共同说明，优先级排序能力是智能体提升决策和执行力的基础。
实战代码示例
以下代码演示了如何用LangChain 构建一个项目经理智能体。该智能体可自动创建、
排序并分配任务，展示了大语言模型结合自定义工具实现项目管理自动化的应用。
优先级排序模式示例
import os
import asyncio
from typing import List, Optional, Dict, Type
from dotenv import load_dotenv
from pydantic import BaseModel, Field
from langchain_core.prompts import ChatPromptTemplate
from langchain_core.tools import Tool
from langchain_openai import ChatOpenAI
from langchain.agents import AgentExecutor, create_react_agent
from langchain.memory import ConversationBufferMemory
# ‑‑‑ 0. 配置与初始化‑‑‑
# 从.env 文件加载OPENAI_API_KEY。
load_dotenv()
# ChatOpenAI 客户端自动读取环境变量中的API key。
llm = ChatOpenAI(temperature=0.5, model="gpt‑4o‑mini")

第20 章：优先级排序
# ‑‑‑ 1. 任务管理系统‑‑‑
class Task(BaseModel):
"""系统中的单个任务。"""
id: str
description: str
priority: Optional[str] = None
# P0, P1, P2
assigned_to: Optional[str] = None # 工作人员姓名
class SuperSimpleTaskManager:
"""高效且健壮的内存任务管理器。"""
def __init__(self):
# 使用字典实现O(1) 查找、更新和删除。
self.tasks: Dict[str, Task] = {}
self.next_task_id = 1
def create_task(self, description: str) ‑> Task:
"""创建并存储新任务。"""
task_id = f"TASK‑{self.next_task_id:03d}"
new_task = Task(id=task_id, description=description)
self.tasks[task_id] = new_task
self.next_task_id += 1
print(f"DEBUG: 创建任务‑ {task_id}: {description}")
return new_task
def update_task(self, task_id: str, **kwargs) ‑> Optional[Task]:
"""使用Pydantic 的model_copy 安全更新任务。"""
task = self.tasks.get(task_id)
if task:
update_data = {k: v for k, v in kwargs.items() if v is not None}
updated_task = task.model_copy(update=update_data)
self.tasks[task_id] = updated_task
print(f"DEBUG: 任务{task_id} 更新为{update_data}")
return updated_task
print(f"DEBUG: 未找到任务{task_id}，无法更新。")
return None
def list_all_tasks(self) ‑> str:
"""列出系统中的所有任务。"""
if not self.tasks:
return "系统中暂无任务。"
task_strings = []
for task in self.tasks.values():
task_strings.append(
f"ID: {task.id}, 描述：'{task.description}', "
f"优先级：{task.priority or 'N/A'}, "
f"分配给：{task.assigned_to or 'N/A'}"
)
return "当前任务列表：\n" + "\n".join(task_strings)
task_manager = SuperSimpleTaskManager()

实战代码示例
# ‑‑‑ 2. 项目经理Agent 工具‑‑‑
# 使用Pydantic 模型定义工具参数，提升校验和可读性。
class CreateTaskArgs(BaseModel):
description: str = Field(description="任务的详细描述。")
class PriorityArgs(BaseModel):
task_id: str = Field(description="要更新的任务ID，例如'TASK‑001'。")
priority: str = Field(description="优先级，必须为'P0'、'P1' 或'P2'。")
class AssignWorkerArgs(BaseModel):
task_id: str = Field(description="要更新的任务ID，例如'TASK‑001'。")
worker_name: str = Field(description="分配任务的工作人员姓名。")
def create_new_task_tool(description: str) ‑> str:
"""根据描述创建新项目任务。"""
task = task_manager.create_task(description)
return f"已创建任务{task.id}: '{task.description}'。"
def assign_priority_to_task_tool(task_id: str, priority: str) ‑> str:
"""为指定任务分配优先级（P0、P1、P2）。"""
if priority not in ["P0", "P1", "P2"]:
return "优先级无效，必须为P0、P1 或P2。"
task = task_manager.update_task(task_id, priority=priority)
return f"已为任务{task.id} 分配优先级{priority}。" if task else f"未找到任务{task_id}。"
def assign_task_to_worker_tool(task_id: str, worker_name: str) ‑> str:
"""将任务分配给指定工作人员。"""
task = task_manager.update_task(task_id, assigned_to=worker_name)
return f"已将任务{task.id} 分配给{worker_name}。" if task else f"未找到任务{task_id}。"
# 项目经理Agent 可用的所有工具
pm_tools = [
Tool(
name="create_new_task",
func=create_new_task_tool,
description="首先用于创建新任务并获取任务ID。",
args_schema=CreateTaskArgs
),
Tool(
name="assign_priority_to_task",
func=assign_priority_to_task_tool,
description="任务创建后用于分配优先级。",
args_schema=PriorityArgs
),
Tool(
name="assign_task_to_worker",
func=assign_task_to_worker_tool,
description="任务创建后用于分配给指定工作人员。",
args_schema=AssignWorkerArgs
),
Tool(
name="list_all_tasks",
func=task_manager.list_all_tasks,
description="用于列出所有当前任务及状态。"

第20 章：优先级排序
),
]
# ‑‑‑ 3. 项目经理Agent 定义‑‑‑
pm_prompt_template = ChatPromptTemplate.from_messages([
("system", """你是一名专注的项目经理LLM Agent，目标是高效管理项目任务。
当收到新任务请求时，请按以下步骤操作：
1. 首先使用`create_new_task` 工具创建任务并获取`task_id`。
2. 分析用户请求，判断是否提及优先级或分配人员。
‑ 如果提到优先级（如“紧急”、“ASAP”、“关键”），映射为P0，使用`assign_priority_to_task`。
‑ 如果提到工作人员，则使用`assign_task_to_worker`。
3. 如信息（优先级、分配人员）缺失，需合理默认分配（如优先级设为P1，分配给'Worker A'）。
4. 任务处理完毕后，使用`list_all_tasks` 展示最终状态。
可用工作人员：'Worker A'、'Worker B'、'Review Team'
优先级：P0（最高）、P1（中）、P2（最低）
"""),
("placeholder", "{chat_history}"),
("human", "{input}"),
("placeholder", "{agent_scratchpad}")
])
# 创建Agent 执行器
pm_agent = create_react_agent(llm, pm_tools, pm_prompt_template)
pm_agent_executor = AgentExecutor(
agent=pm_agent,
tools=pm_tools,
verbose=True,
handle_parsing_errors=True,
memory=ConversationBufferMemory(memory_key="chat_history", return_messages=True)
)
# ‑‑‑ 4. 简单交互流程‑‑‑
async def run_simulation():
print("‑‑‑ 项目经理Agent 模拟‑‑‑")
# 场景1：处理紧急新功能请求
print("\n[用户请求] 需要ASAP 实现新的登录系统，分配给Worker B。")
await pm_agent_executor.ainvoke({"input": "创建一个实现新登录系统的任务。很紧急，分配给Worker B。
"})
,!
print("\n" + "‑"*60 + "\n")
# 场景2：处理不太紧急的内容更新请求
print("[用户请求] 需要审核营销网站内容。")
await pm_agent_executor.ainvoke({"input": "管理一个新任务：审核营销网站内容。"})
print("\n‑‑‑ 模拟结束‑‑‑")
# 运行模拟
if __name__ == "__main__":
asyncio.run(run_simulation())

一图速览
该代码实现了一个基于Python 和LangChain 的简单任务管理系统，模拟了由大语言模
型驱动的项目经理智能体。
系统通过
SuperSimpleTaskManager 类在内存中高效管理任务，采用字典结构实现快
速数据检索。每个任务由Task
Pydantic 模型表示，包含唯一标识、描述文本、可选
优先级（P0、P1、P2）和可选分配人员。内存使用量取决于任务类型、工作人员数量
等因素。任务管理器提供任务创建、修改和查询方法。
智能体通过一组工具与任务管理器交互。这些工具包括新任务创建、优先级分配、任务
分配和任务列表查询。每个工具都封装了与
SuperSimpleTaskManager 实例的交互，
参数采用
Pydantic 模型定义，确保数据校验。
AgentExecutor 配置了语言模型、工具集和对话记忆组件，保证上下文连贯。通过
ChatPromptTemplate 明确智能体的项目管理行为：先创建任务，再根据需求分配优先
级和人员，最后输出任务列表。对于缺失信息，提示中规定默认分配优先级为P1，分
配给‘Worker A'。
代码包含一个异步模拟函数（
run_simulation ），演示智能体的实际操作流程。模拟
场景包括处理紧急任务和常规任务，智能体的操作和逻辑通过
verbose=True 输出到
控制台。
一图速览
是什么：智能体在复杂环境下面临大量潜在行动、目标冲突和有限资源。如果没有明确
的决策方法，智能体容易低效甚至失效，导致操作延迟或无法完成主要目标。核心挑战
是管理众多选择，确保智能体有目的、合理地行动。
为什么：优先级排序模式为此类问题提供标准化解决方案，让智能体能够对任务和目标
进行排序。通过设定紧急性、重要性、依赖关系、资源成本等明确标准，智能体评估每
个潜在行动，确定最关键、最及时的方案。这种智能体能力让系统能动态适应变化，有
效管理有限资源，专注于最高优先级事项，使行为更智能、更稳健、更具战略性。
经验法则：当智能体系统需在资源受限、任务或目标冲突的动态环境下自主管理多项任
务时，应采用优先级排序模式。
视觉摘要：

第20 章：优先级排序
图1：优先级排序设计模式
关键要点
・优先级排序让智能体在复杂多元环境下高效运作。
・智能体通过紧急性、重要性、依赖关系等标准评估和排序任务。
・动态优先级调整使智能体能实时响应环境变化。
・优先级排序可应用于战略目标和即时战术决策等多个层级。
・有效的优先级排序提升智能体效率和操作稳健性。
总结
综上，优先级排序模式是高效智能体的基石，使系统能够有目的地应对动态环境的复杂
挑战。它让智能体能自主评估众多冲突任务和目标，合理分配有限资源，做出理性决
策。这种智能体能力不仅仅是简单执行任务，更让系统具备主动、战略性的决策能力。
通过权衡紧急性、重要性和依赖关系，智能体展现出类似人类的高级推理过程。
动态优先级调整是智能体行为的关键特性，使智能体能根据实时变化自主调整关注重
点。正如代码示例所示，智能体能理解模糊请求，自主选择并使用合适工具，合理安排
行动顺序以达成目标。这种自我管理能力是智能体系统与普通自动化脚本的本质区别。
最终，掌握优先级排序是打造稳健、智能的智能体在复杂真实场景下可靠运行的基础。

参考文献
参考文献
・人工智能在项目管理中的安全性研究：以AI 驱动的项目调度与资源分配为例‑
irejournals.com
・敏捷软件项目管理中的AI 决策支持系统：提升风险规避与资源分配‑ mdpi.com