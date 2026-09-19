import psycopg, os

# 用法：PGP=<密码> python3 init_remote_db.py
# 在外部 Postgres 上：
#   1) 以默认库 postgres 连接，创建 mem0 与 mem0_app 两个库（若不存在）
#   2) 在 mem0 库启用 vector 扩展
pw = os.environ["PGP"]


def db_exists(conn, name):
    cur = conn.cursor()
    cur.execute("SELECT 1 FROM pg_database WHERE datname=%s", (name,))
    r = cur.fetchone()
    cur.close()
    return r is not None


admin = psycopg.connect(
    host="192.168.100.80", port=5432, user="postgres",
    password=pw, dbname="postgres", autocommit=True,
)
for db in ("mem0", "mem0_app"):
    if not db_exists(admin, db):
        admin.cursor().execute(f'CREATE DATABASE "{db}"')
        print(f"{db}: created")
    else:
        print(f"{db}: exists")
admin.close()

main = psycopg.connect(
    host="192.168.100.80", port=5432, user="postgres",
    password=pw, dbname="mem0", autocommit=True,
)
cur = main.cursor()
cur.execute("CREATE EXTENSION IF NOT EXISTS vector")
cur.close()
main.close()
print("vector extension: ok")
