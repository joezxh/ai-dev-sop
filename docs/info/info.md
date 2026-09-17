# 网警同款开盒思路，查人查公司查设备，五个免费开源工具

> 来源：X / Twitter 帖子（该帖正文为一条链接，指向作者的 X 长文，本文为其长文全文）

| 项目 | 内容 |
| --- | --- |
| 作者 | 老杨啊 \| AI产品商业化（[@yhslgg](https://x.com/yhslgg)） |
| 发布时间 | 2026-06-20 12:56:42 UTC（北京时间 2026-06-20 20:56:42） |
| 原帖链接 | https://x.com/yhslgg/status/2068317116831510838 |
| 长文链接 | https://x.com/i/article/2068314177245184000 |
| 互动数据 | 7,502 喜欢 · 138 回复 |

![封面：网警同款开盒思路，查人查公司查设备，五个免费开源工具](images/00-cover.jpg)

最近连续分享了好几个 OSINT 工具，今天就给兄弟们讲讲怎么组合用。

今天一次说清楚——五个工具，覆盖你能想到的几乎所有查人、查公司、查设备场景，直接开盒任何人，

本文只讲思路和介绍工具及YY使用场景；不提倡实际应用。

这套东西不提倡查别人，希望大家用来保护自己。

## 先说五个工具各干什么

![1. 先说五个工具各干什么](images/01-five-tools-overview.jpg)

Blackbird（6300 Star）——用户名和邮箱都能查，接入 600+ 平台，查完免费调 AI 生成人物画像，PDF 一键导出。新手入门首选。

Maigret（33000 Star）——用户名查人的专精版，覆盖 3000+ 平台，核心是递归追踪——发现你在 A 平台还有另一个马甲，自动拿这个马甲继续搜，顺着线挖到底。

SpiderFoot（18000 Star）——重炮。手机号、邮箱、域名、IP 全吃，200+ 模块同时出动，能查数据泄露、暗网、子域名，出可视化关系图谱。

theHarvester（16500 Star）——给你一个域名，自动从谷歌、必应、LinkedIn、DNS 记录等 40+ 个数据源里，把这家公司的员工邮箱、子域名、IP、URL 批量扒出来。查公司专用。

Shodan Python（2800 Star）——Shodan 是"设备版谷歌"，全网扫描暴露在公网上的摄像头、路由器、服务器、工业控制系统。这个库让你用代码自动化查询 Shodan 数据库。

## 真实场景，五个工具怎么用

## 场景一：测谎另一半。

![2. 场景一：测谎另一半。](images/02-scenario1-partner-check.jpg)

另一半说"我不玩社交软件"，你只知道他/她的常用昵称。

第一步 Blackbird：昵称扔进去，600 个平台扫一遍，陌陌、Soul、探探、Tinder 有没有账号，AI 直接出画像。

第二步 Maigret 接力：Blackbird 找到的账号里，可能暴露了另一个用户名。拿新用户名丢进 Maigret，3000 个平台再扫一圈，递归追踪到底。

五分钟基本有答案了，爱TA，还是啪啪打脸TA，就看命了。

## 场景二：查情敌/查小三 / 查陌生人

![3. 场景二：查情敌/查小三 / 查陌生人](images/03-scenario2-username-or-email.jpg)

只有一个用户名或邮箱。

先用 Blackbird 快速出全貌，再用 Maigret 深挖递归，把这个人在网上所有马甲、发言记录、混过的圈子全翻出来。

再拿邮箱丢进 SpiderFoot，查有没有在历史数据泄露里出现，看看他/她有没有被拖过库。

## 场景三：查一家公司，商业尽调

![4. 场景三：查一家公司，商业尽调](images/04-scenario3-company-duediligence.jpg)

准备跟一家公司合作，想先摸底。

第一步 theHarvester：输入对方域名，自动从 LinkedIn、谷歌、DNS 里扒出公司员工邮箱列表，能看到邮箱格式规律（比如 firstname.lastname@company.com），顺带出子域名和 IP 列表。

第二步 SpiderFoot：拿域名深挖，扫全部子域名和服务器信息，查有没有出现在数据泄露事件里，往暗网搜一圈看有没有公司数据被挂牌出售。

第三步 Shodan：拿 IP 段扔进 Shodan，看这家公司有没有把数据库、后台管理界面、监控摄像头直接暴露在公网——这个细节能看出对方的技术水平和安全意识。

合作之前前花半小时，省的事多了去。

## 场景四：查自己公司的安全及舆情管理

![5. 场景四：查自己公司的安全及舆情管理](images/05-scenario4-own-company-security.jpg)

很多公司不知道自己在网上暴露了什么。

theHarvester 查域名，看对外暴露的邮箱有多少、子域名有多少；Shodan 查 IP 段，看有没有服务器端口、摄像头、内部系统直接对公网开着口子。

很多小公司的 NAS、摄像头、打印机，配置没改，直接就能在 Shodan 上搜到，连密码都是出厂默认值——自己查一遍，能查出来很多意想不到的漏洞。

## 场景五：查竞争对手的技术架构

想知道竞争对手用了什么技术栈、服务器在哪、用了哪些云服务。

theHarvester 扒出子域名列表，能看出对方的产品线分布；Shodan 查 IP，能看出用了什么服务器软件、哪个版本、部署在哪个云上。

技术选型、团队规模、基础设施——这些以前要靠内线打听，现在公开信息就能拼出七七八八。

## 场景六：查自己的数字痕迹，清理信息

把自己的常用用户名、邮箱分别跑一遍 Blackbird 和 Maigret，看有多少你忘掉的老账号还挂在各平台上。

再把邮箱丢进 SpiderFoot，看有没有在数据泄露事件里出现过，被哪次拖库波及。

知道暴露了什么，才能有针对性地清理、注销、换密码。

## 五个工具，怎么搭配

![6. 五个工具，怎么搭配](images/06-tools-combination-flow.jpg)

> 拿到用户名   →  Blackbird（快速全貌）→ Maigret（递归深挖）
> 拿到邮箱     →  Blackbird（账号+AI画像）→ SpiderFoot（泄露+暗网）
> 拿到域名     →  theHarvester（邮箱/子域名批量采集）→ SpiderFoot（深挖）
> 拿到 IP      →  Shodan（设备和服务暴露情况）→ SpiderFoot（关联情报）
> 什么都有     →  全部上，交叉验证

原则只有一条：先用最快的出全貌，再用最深的往下挖。

## 安装，一人一行命令

> # Blackbird
> git clone https://github.com/p1ngul1n0/blackbird && pip install -r requirements.txt
> # Maigret
> pip install maigret
> # SpiderFoot
> git clone https://github.com/smicallef/spiderfoot && pip install -r requirements.txt
> # theHarvester
> git clone https://github.com/laramies/theHarvester && uv sync
> # Shodan Python
> pip install shodan

Blackbird、Maigret、SpiderFoot 不需要 API Key，开箱即用。

theHarvester 部分数据源需要 Key（谷歌、Bing 等），但基础功能免费够用。

Shodan 需要注册一个免费账号拿 API Key，免费额度日常够用。

## 最后说一句

这五个工具放在一起，代表的是一种思路：

一个人、一家公司，在网上能藏住的东西，远比他们以为的少得多。

你以为删掉的账号，以为没人知道的马甲，以为没有暴露的服务器——只要存在过，大概率能找回来。

再次声明。这套东西不提倡查别人，希望大家用来保护自己。

知道自己暴露了什么，是第一步。

五个项目链接：

- github.com/p1ngul1n0/blackbird
- github.com/soxoj/maigret
- github.com/smicallef/spiderfoot
- github.com/laramies/theHarvester
- github.com/achillean/shodan-python

好东西转给需要的兄弟。🚀

\#OSINT #开源工具 #老杨啊分享

---

## 项目链接汇总

- [Blackbird](https://github.com/p1ngul1n0/blackbird)
- [Maigret](https://github.com/soxoj/maigret)
- [SpiderFoot](https://github.com/smicallef/spiderfoot)
- [theHarvester](https://github.com/laramies/theHarvester)
- [Shodan Python](https://github.com/achillean/shodan-python)

> 免责声明：本文由原作者发布，内容为 OSINT 工具的介绍与使用思路。**仅用于自我防护与安全评估，请勿用于侵犯他人隐私或非法用途。**
