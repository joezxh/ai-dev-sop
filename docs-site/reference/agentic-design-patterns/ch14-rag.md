# 第14章：知识检索（RAG）

第14 章：知识检索（RAG）
大语言模型（LLM）在生成类人文本方面展现出强大能力，但其知识库通常仅限于训练
数据，无法访问实时信息、企业内部数据或高度专业化的细节。知识检索（RAG，
Retrieval Augmented Generation）正是为了解决这一局限。RAG 让LLM 能够访问并
集成外部、最新、特定场景的信息，从而提升输出的准确性、相关性和事实基础。
对于智能体而言，这一能力至关重要，因为它能让智能体的行为和响应基于实时、可验
证的数据，而不仅仅是静态训练内容。智能体能够准确执行复杂任务，例如查阅最新公
司政策来回答具体问题，或在下单前核查当前库存。通过集成外部知识，RAG 让智能体
从简单的对话者转变为高效的数据驱动工具，能够完成有意义的工作。
知识检索（RAG）模式概述
知识检索（RAG）模式显著增强了LLM 的能力，使其在生成响应前能够访问外部知识
库。与仅依赖内部预训练知识不同，RAG 允许LLM“查找”信息，类似于人类查阅书籍
或搜索互联网。这一过程让LLM 能够提供更准确、最新、可验证的答案。
当用户向采用RAG 的AI 系统提出问题或请求时，查询不会直接发送给LLM，而是先在
庞大的外部知识库（如文档库、数据库或网页）中检索相关信息。这种检索不仅仅是关
键词匹配，而是“语义搜索”，理解用户意图和词语背后的含义。系统会提取最相关的
信息片段（chunk），并将这些内容“增强”到原始提示中，形成更丰富、更有信息量的
查询。最终，这个增强后的提示被送入LLM，借助额外的上下文，LLM 能生成既流畅自
然又有事实依据的响应。
RAG 框架带来了多项显著优势。它让LLM 能够访问最新信息，突破静态训练数据的限
制；通过以可验证数据为基础，降低了“幻觉”（虚假信息）的风险；还能利用企业内
部文档或Wiki 等专业知识。一个重要优势是能够提供“引用”，明确指出信息来源，提
升AI 响应的可信度和可验证性。
要全面理解RAG 的工作原理，需掌握几个核心概念（见图1）：
嵌入（Embeddings）：在LLM 语境下，嵌入是文本（如词语、短语或文档）的数值表
示，通常为向量（数字列表）。其核心思想是用数学空间表达语义和文本间的关系。含
义相近的词或短语，其嵌入在向量空间中距离更近。例如，“cat”可能是(2, 3)，而
“kitten”则在(2.1, 3.1) 附近；“car”则远在(8, 1)。实际嵌入空间维度远高于二维，能

第14 章：知识检索（RAG）
细致刻画语言的语义。
文本相似度：指两段文本的相似程度，可分为表层（词汇重叠）和深层（语义）。在
RAG 中，文本相似度用于在知识库中找到与用户查询最相关的信息。例如，“What is
the capital of France?”和“Which city is the capital of France?”虽措辞不同，但表达
同一问题。优秀的相似度模型会识别并赋予高分，这通常通过文本嵌入计算。
语义相似度与距离：语义相似度关注文本的含义和上下文，而非仅词汇。语义距离则是
其反向指标。RAG 的语义搜索就是寻找与用户查询语义距离最小的文档。例如，“a
furry feline companion”和“a domestic cat”虽词汇不同，但表达同一概念，嵌入距
离很近。正是这种“智能搜索”让RAG 能在措辞不同的情况下找到相关信息。
图1：RAG 核心概念：分块、嵌入和向量数据库
文档分块（Chunking）：分块是将大文档拆分为更小、更易处理的片段。RAG 系统无法
将整本大文档输入LLM，而是处理这些小块。分块策略对于保留信息的上下文和意义至
关重要。例如，将50 页用户手册按章节、段落或句子分块，“故障排查”与“安装指
南”分别为独立块。用户提问时，RAG 能检索最相关的块而非整本手册，提高检索效率
和响应针对性。分块后，RAG 需采用检索技术找到最相关内容，主要方法是向量搜索
（利用嵌入和语义距离）。传统BM25 算法则基于关键词频率，不理解语义。为兼顾两
者，常用混合检索，将BM25 的精确匹配与语义搜索结合，实现更强大和准确的检索。

知识检索（RAG）模式概述
向量数据库：向量数据库专为高效存储和查询嵌入设计。文档分块并转为嵌入后，这些
高维向量存入向量数据库。传统关键词检索只能找到包含查询词的文档，无法理解语
义。例如，“furry feline companion”与“cat”无法关联。向量数据库则专注于语义搜
索，将文本以数值向量存储，能根据概念意义检索结果。用户查询也转为向量，数据库
用高效算法（如HNSW）在海量向量中快速查找“最接近”的含义。这一技术远胜于
RAG，能发现措辞完全不同但语义相关的内容。主流实现包括Pinecone、Weaviate、
Chroma DB、Milvus、Qdrant 等，甚至Redis、Elasticsearch、Postgres（pgvector 扩
展）也可支持向量检索。底层检索机制常用FAISS、ScaNN 等库，保证系统高效。
RAG 的挑战：尽管强大，RAG 也面临诸多挑战。主要问题是答案所需信息可能分散在
多个块或文档中，检索器难以获取完整上下文，导致答案不准确或不完整。系统效果高
度依赖分块和检索质量，若检索到无关块则会引入噪声，干扰LLM。如何有效整合可能
矛盾的信息也是难题。此外，RAG 需将整个知识库预处理并存入专用数据库（如向量或
图数据库），这是一项庞大工程，且需定期同步以保持最新，尤其是企业Wiki 等动态内
容。整个流程会影响性能，增加延迟、运维成本和最终提示的Token 数量。
总之，RAG 模式让AI 更加智能和可靠。通过在生成流程中无缝集成外部知识检索，
RAG 克服了LLM 的核心局限。嵌入和语义相似度等基础技术，结合关键词和混合检索，
让系统能智能地找到相关信息，并通过分块实现高效管理。整个检索过程由专用向量数
据库驱动，能高效存储和查询海量嵌入。尽管在碎片化或矛盾信息检索上仍有挑战，
RAG 让LLM 能生成更具上下文和事实依据的答案，提升AI 的可信度和实用性。
图RAG（GraphRAG）：GraphRAG 是RAG 的高级形式，利用知识图谱而非简单向量数
据库进行信息检索。它通过遍历知识图谱中实体（节点）间的显式关系（边）来回答复
杂问题，能整合分散在多个文档的信息，弥补传统RAG 的不足。通过理解数据间的连
接，GraphRAG 能提供更具上下文和细致度的答案。
应用场景包括复杂金融分析、企业与市场事件关联、科学研究（如基因与疾病关系发
现）。主要缺点是构建和维护高质量知识图谱的复杂性、成本和专业要求极高，系统灵
活性较低，延迟也可能高于简单向量检索。系统效果完全依赖底层图结构的质量和完整
性。因此，GraphRAG 在需要深度、关联洞察时表现优异，但实现和维护成本较高。
智能体RAG：智能体RAG 是该模式的进化版，引入推理和决策层，显著提升信息提取
的可靠性。除了检索和增强外，智能体作为关键把关者和知识精炼者主动参与。智能体
不被动接受初步检索结果，而是主动审查其质量、相关性和完整性，具体场景如下：
首先，智能体擅长反思和来源验证。例如，用户问“公司远程办公政策是什么？”，标准
RAG 可能同时检索到2020 年博客和2025 年官方政策。智能体会分析文档元数据，识
别2025 政策为最新权威来源，丢弃过时博客，仅将正确内容送入LLM，确保答案准确。

第14 章：知识检索（RAG）
图2：智能体RAG 引入推理Agent，主动评估、整合和精炼检索信息，确保最终响应更
准确可信。
其次，智能体能调和知识冲突。比如财务分析师问“Alpha 项目Q1 预算是多少？”，系
统检索到两份文件：初步方案为€50,000，最终报告为€65,000。智能体RAG 会识别矛
盾，优先采用财务报告，确保答案基于最可靠数据。
第三，智能体能多步推理，综合复杂答案。例如用户问“我们产品的功能和定价与竞争
对手X 有何区别？”，智能体会拆分为子查询，分别检索自身产品功能、定价、竞争对手
功能和定价，最后综合成结构化对比内容送入LLM，实现全面响应。
第四，智能体能识别知识空缺并调用外部工具。比如用户问“昨天新产品发布后市场的
即时反应如何？”，智能体检索内部知识库（每周更新）未找到相关信息，识别空缺后可
调用实时Web 搜索API，获取最新新闻和社交媒体舆情，用于生成最新答案，突破静态
数据库限制。
智能体RAG 的挑战：虽然强大，智能体层也带来复杂性和成本的大幅提升。设计、实
现和维护智能体的决策逻辑及工具集成需大量工程投入，计算开销也更高。复杂性可能
导致延迟增加，智能体的反思、工具调用和多步推理比直接检索耗时更多。此外，智能
体本身也可能成为新错误源，推理失误可能导致陷入无用循环、误解任务或错误丢弃相
关信息，最终影响响应质量。
总结：智能体RAG 是标准检索模式的高级演化，将其从被动数据管道转变为主动问题
解决框架。通过嵌入推理层，智能体能评估来源、调和冲突、拆解复杂问题并调用外部

实践应用与场景
工具，大幅提升答案的可靠性和深度。虽然带来系统复杂性、延迟和成本的权衡，但显
著增强了AI 的可信度和能力。
实践应用与场景
知识检索（RAG）正在改变LLM 在各行业的应用方式，提升其提供准确、相关响应的
能力。
应用场景包括：
・企业搜索与问答：企业可开发内部聊天机器人，利用HR 政策、技术手册、产品规格
等内部文档回答员工问题。RAG 系统会提取相关文档片段辅助LLM 响应。
・客户支持与服务台：基于RAG 的系统可通过产品手册、FAQ、工单等信息，为客户
提供精准一致的答复，减少人工介入。
・个性化内容推荐：RAG 能根据用户偏好或历史行为，语义检索相关内容（文章、产
品），实现更相关的推荐，而非简单关键词匹配。
・新闻与时事摘要：LLM 可集成实时新闻源，用户提问时，RAG 检索最新文章，让
LLM 生成最新摘要。
通过集成外部知识，RAG 让LLM 超越简单交流，成为知识处理系统。
实践代码示例（ADK）
以下用三个示例说明知识检索（RAG）模式。
首先，展示如何用Google Search 实现RAG，让LLM 基于搜索结果进行知识增强。
Google Search 工具是内置检索机制的直接例子，可增强LLM 的知识。
from google.adk.tools import google_search
from google.adk.agents import Agent
search_agent = Agent(
name="research_assistant",
model="gemini‑2.0‑flash‑exp",
instruction="You help users research topics. When asked, use the Google Search tool",
tools=[google_search]
)
第二，说明如何在Google ADK 中利用Vertex AI RAG 能力。代码演示了通过ADK 初始

第14 章：知识检索（RAG）
化
VertexAiRagMemoryService ，连接Google Cloud Vertex AI RAG Corpus。可配置参
数如
SIMILARITY_TOP_K 和
VECTOR_DISTANCE_THRESHOLD ，分别控制检索结果数量和
语义距离阈值。这样智能体能从指定RAG Corpus 进行可扩展、持久的语义知识检索，
将Google Cloud RAG 功能集成到ADK Agent，支持基于事实的数据响应开发。
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
实践代码示例（LangChain）
第三，完整演示LangChain 的RAG 实现。
LangChain RAG 实现
import os
import requests
from typing import List, Dict, Any, TypedDict
from langchain_community.document_loaders import TextLoader
from langchain_core.documents import Document
from langchain_core.prompts import ChatPromptTemplate
from langchain_core.output_parsers import StrOutputParser
from langchain_community.embeddings import OpenAIEmbeddings
from langchain_community.vectorstores import Weaviate
from langchain_openai import ChatOpenAI
from langchain.text_splitter import CharacterTextSplitter
from langchain.schema.runnable import RunnablePassthrough
from langgraph.graph import StateGraph, END
import weaviate
from weaviate.embedded import EmbeddedOptions
import dotenv
dotenv.load_dotenv()
url = "https://github.com/langchain‑ai/langchain/blob/master/docs/docs/how_to/state_of_the_union.t c
xt"
,!

实践代码示例（LangChain）
res = requests.get(url)
with open("state_of_the_union.txt", "w") as f:
f.write(res.text)
loader = TextLoader('./state_of_the_union.txt')
documents = loader.load()
text_splitter = CharacterTextSplitter(chunk_size=500, chunk_overlap=50)
chunks = text_splitter.split_documents(documents)
client = weaviate.Client(
embedded_options = EmbeddedOptions()
)
vectorstore = Weaviate.from_documents(
client = client,
documents = chunks,
embedding = OpenAIEmbeddings(),
by_text = False
)
retriever = vectorstore.as_retriever()
llm = ChatOpenAI(model_name="gpt‑3.5‑turbo", temperature=0)
class RAGGraphState(TypedDict):
question: str
documents: List[Document]
generation: str
def retrieve_documents_node(state: RAGGraphState) ‑> RAGGraphState:
question = state["question"]
documents = retriever.invoke(question)
return {"documents": documents, "question": question, "generation": ""}
def generate_response_node(state: RAGGraphState) ‑> RAGGraphState:
question = state["question"]
documents = state["documents"]
template = """You are an assistant for question‑answering tasks.
Use the following pieces of retrieved context to answer the question.
If you don't know the answer, just say that you don't know.
Use three sentences maximum and keep the answer concise.
Question: {question}
Context: {context}
Answer:
"""
prompt = ChatPromptTemplate.from_template(template)
context = "\n\n".join([doc.page_content for doc in documents])
rag_chain = prompt ￨llm ￨StrOutputParser()
generation = rag_chain.invoke({"context": context, "question": question})
return {"question": question, "documents": documents, "generation": generation}
workflow = StateGraph(RAGGraphState)
workflow.add_node("retrieve", retrieve_documents_node)

第14 章：知识检索（RAG）
workflow.add_node("generate", generate_response_node)
workflow.set_entry_point("retrieve")
workflow.add_edge("retrieve", "generate")
workflow.add_edge("generate", END)
app = workflow.compile()
if __name__ == "__main__":
print("\n‑‑‑ Running RAG Query ‑‑‑")
query = "What did the president say about Justice Breyer"
inputs = {"question": query}
for s in app.stream(inputs):
print(s)
print("\n‑‑‑ Running another RAG Query ‑‑‑")
query_2 = "What did the president say about the economy?"
inputs_2 = {"question": query_2}
for s in app.stream(inputs_2):
print(s)
上述Python 代码展示了用LangChain 和LangGraph 实现的RAG 流程。首先从文本文
档创建知识库，分块并转为嵌入，存入Weaviate 向量库，实现高效检索。LangGraph
的StateGraph 管理检索和生成两个核心节点：retrieve_documents_node 用于根据用
户输入检索相关文档块，generate_response_node 用于结合检索内容和预设提示模
板，调用OpenAI LLM 生成响应。app.stream 方法可运行RAG 查询，展示系统生成上
下文相关输出的能力。
一图速览
是什么：LLM 虽有强大文本生成能力，但受限于训练数据，知识是静态的，无法包含实
时或专有数据，导致响应可能过时、不准确或缺乏特定场景所需的上下文，影响其在需
最新、事实答案场景下的可靠性。
为什么：RAG 模式通过连接LLM 与外部知识源，提供标准化解决方案。收到查询后，
系统先从指定知识库检索相关信息片段，再将这些片段附加到原始提示，丰富上下文，
最后送入LLM，生成准确、可验证、基于外部数据的响应。此过程让LLM 从“闭卷”推
理者变为“开卷”推理者，显著提升实用性和可信度。
经验法则：当需要LLM 基于最新、专有或训练数据之外的信息回答问题或生成内容时，
建议采用此模式。适用于构建内部文档问答系统、客户支持机器人，以及需可验证、带
引用的事实型响应应用。
视觉摘要

一图速览
图3：知识检索模式：智能体查询结构化数据库获取信息
图4：知识检索模式：智能体从互联网查找并综合信息响应用户问题

第14 章：知识检索（RAG）
关键要点
・知识检索（RAG）让LLM 能访问外部、最新、专有信息。
・包含检索（搜索知识库相关片段）和增强（将片段加入LLM 提示）两步。
・RAG 帮助LLM 克服训练数据过时、减少“幻觉”，实现领域知识集成。
・RAG 支持可归因答案，响应基于检索来源。
・GraphRAG 利用知识图谱理解信息间关系，能回答需多源综合的复杂问题。
・智能体RAG 通过智能体主动推理、验证和精炼外部知识，确保答案更准确可靠。
・实践应用涵盖企业搜索、客户支持、法律检索、个性化推荐等场景。
总结
总之，RAG 通过连接LLM 与外部、最新数据源，解决了其静态知识的核心局限。流程
包括先检索相关信息片段，再增强用户提示，使LLM 能生成更准确、具上下文的响应。
嵌入、语义搜索和向量数据库等基础技术让系统能基于语义而非关键词查找信息。通过
以可验证数据为基础，RAG 显著减少事实错误，支持专有信息使用，提升可信度。
智能体RAG 进一步引入推理层，主动验证、整合和综合检索知识，提升可靠性。
GraphRAG 则利用知识图谱，能处理复杂、关联性强的问题。智能体能解决信息冲突、
多步查询、调用外部工具，极大提升响应深度和可信度。虽然这些高级方法增加了系统
复杂性和延迟，但已在企业搜索、客户支持、个性化内容等领域带来变革。RAG 是让AI
更智能、可靠和实用的关键模式，最终让LLM 从“闭卷对话者”变为强大的“开卷推理
工具”。
参考文献
・Lewis, P., 等（2020）Retrieval‑Augmented Generation for Knowledge‑Intensive
NLP Tasks ‑ arxiv.org
・Google AI for Developers Documentation. Retrieval Augmented Generation
Overview ‑ cloud.google.com
・Retrieval‑Augmented Generation with Graphs (GraphRAG) ‑ arxiv.org
・LangChain 和LangGraph：Leonie Monigatti,“Retrieval‑Augmented Generation
(RAG): From Theory to LangChain Implementation”‑ medium.com

参考文献
・Google Cloud Vertex AI RAG Corpus ‑ cloud.google.com