// scripts/scaffold-models.mjs — information_schema 反推生成 GORM 桩（DB-First，家族 ADR-0025 口径）
//
// 设计（REQ-2026-003；springboot scaffold-entities / laravel scaffold-models 同构第五链）：
// - 直读 information_schema + pg_catalog，emit Go struct 到 internal/gen/entity/<table>.go
// - 简化输出：TableName 显式 + 列 tag（gorm:"column:<snake>;type:<pg>"）全显式；
//   复合 PK 全列 primaryKey；手写 repository/常量/领域方法写在 internal/impl/ 层叠加
// - 未知 PG 类型 throw 不静默（fail-loud）；输出按表名/列序稳定排序、无时间戳
// - 尾部漂移护栏：生成区与 git HEAD 不一致（含 untracked）→ exit 2 FATAL
//   （首跑 untracked FATAL 属预期——git add + commit 后重跑核绿）
//
// 用法：DATABASE_URL=postgresql://postgres:postgres@127.0.0.1:54321/saas_dev \
//         node scripts/scaffold-models.mjs
// 前提：shared 仓已 db:migrate（saas_dev schema 与 src/db/schema.ts 一致）
// DATABASE_URL 必填显式传入（CLAUDE.md §2 禁兜底）；suite checker strict 经
// subprocess env 注入（门禁侧契约常量）。

import { createRequire } from "node:module";
import { resolve, dirname } from "node:path";
import { fileURLToPath } from "node:url";
import { mkdirSync, writeFileSync } from "node:fs";
import { execSync } from "node:child_process";

const __dirname = dirname(fileURLToPath(import.meta.url));
const ROOT = resolve(__dirname, "..");

// 借 pg driver（go 仓无 node_modules；走 sibling nextjs，springboot 同款）
const requireFromNext = createRequire(
  resolve(ROOT, "../saas-identity-platform-nextjs/package.json"),
);
const { Client } = requireFromNext("pg");

const DATABASE_URL = process.env.DATABASE_URL;
if (!DATABASE_URL) {
  console.error("[scaffold-models] FATAL: DATABASE_URL 未设（CLAUDE.md §2 禁字面量兜底）");
  console.error("[scaffold-models]        fix: DATABASE_URL='postgresql://postgres:***@127.0.0.1:54321/saas_dev' node scripts/scaffold-models.mjs");
  process.exit(2);
}

const OUTPUT_DIR = resolve(ROOT, "internal/gen/entity");
const ZONE = "internal/gen/entity";

const EXCLUDE_TABLES = new Set([
  // Drizzle tracking — 不是业务表
  "__drizzle_migrations",
  // Rails bookkeeping（REQ-2026-001 期间发现，家族口径同 springboot）：saas_dev 由
  // 本族 rails 栈共用，ActiveRecord 会建自己的 bookkeeping 表——不是家族业务 schema
  "schema_migrations",
  "ar_internal_metadata",
]);

// PG 类型 → Go 类型（GORM tag type: 显式落 PG 型，未知 throw 不静默）
function mapGoType(dataType, udtName, nullable) {
  const wrap = (go, tagType, extraImports = []) => ({
    go: nullable ? `*${go}` : go,
    tagType,
    imports: extraImports,
  });
  switch (dataType) {
    case "uuid":
      return wrap("string", "uuid");
    case "text":
    case "character varying":
    case "character":
      return wrap("string", null);
    case "integer":
      return wrap("int32", null);
    case "smallint":
      return wrap("int16", null);
    case "bigint":
      return wrap("int64", null);
    case "boolean":
      return wrap("bool", null);
    case "jsonb":
      return wrap("[]byte", "jsonb");
    case "timestamp with time zone":
    case "timestamp without time zone":
    case "date":
      return wrap("time.Time", dataType === "date" ? "date" : null, ["time"]);
    case "ARRAY": {
      // udtName 带 _ 前缀表示数组元素类型（如 _text / _uuid）
      const elem = udtName.replace(/^_/, "");
      if (["text", "varchar", "character varying", "uuid"].includes(elem)) {
        const pgArr = elem === "uuid" ? "uuid[]" : "text[]";
        return { go: "pq.StringArray", tagType: pgArr, imports: ["github.com/lib/pq"] };
      }
      throw new Error(`Unsupported array elem udt: ${udtName}`);
    }
    case "USER-DEFINED":
      // enum scaffold 用 string；手写层需要强类型时在 impl 层定义常量
      return wrap("string", udtName);
    default:
      throw new Error(`Unsupported PG type: ${dataType} (${udtName})`);
  }
}

function snakeToCamel(s) {
  return s.replace(/_([a-z0-9])/g, (_, c) => c.toUpperCase());
}

function snakeToPascal(s) {
  const camel = snakeToCamel(s);
  return camel[0].toUpperCase() + camel.slice(1);
}

async function main() {
  const client = new Client({
    connectionString: DATABASE_URL,
    connectionTimeoutMillis: 10000,
  });
  await client.connect();
  console.log(`[scaffold-models] 连接 ${DATABASE_URL.replace(/:[^:@/]+@/, ":***@")}`);

  // 1. 读所有表（稳定序：表名排序）
  const { rows: tables } = await client.query(`
    SELECT table_name
    FROM information_schema.tables
    WHERE table_schema = 'public' AND table_type = 'BASE TABLE'
    ORDER BY table_name
  `);
  const filtered = tables.filter((t) => !EXCLUDE_TABLES.has(t.table_name));
  console.log(`[scaffold-models] 发现 ${filtered.length} 张表`);

  // 2. 复合 PK 表：table_name → PK 列名数组（按 ordinal_position）
  const { rows: pkRows } = await client.query(`
    SELECT tc.table_name, kcu.column_name
    FROM information_schema.table_constraints tc
    JOIN information_schema.key_column_usage kcu
      ON tc.constraint_name = kcu.constraint_name
     AND tc.table_schema = kcu.table_schema
    WHERE tc.table_schema = 'public'
      AND tc.constraint_type = 'PRIMARY KEY'
    ORDER BY tc.table_name, kcu.ordinal_position
  `);
  const pkByTable = {};
  for (const r of pkRows) {
    (pkByTable[r.table_name] ??= []).push(r.column_name);
  }

  for (const { table_name: table } of filtered) {
    const { rows: columns } = await client.query(
      `
      SELECT column_name, data_type, udt_name, is_nullable, ordinal_position
      FROM information_schema.columns
      WHERE table_schema = 'public' AND table_name = $1
      ORDER BY ordinal_position
      `,
      [table],
    );

    const pks = pkByTable[table] ?? [];
    const structName = snakeToPascal(table);
    const imports = new Set();
    const fields = [];

    for (const col of columns) {
      const mapped = mapGoType(col.data_type, col.udt_name, col.is_nullable === "YES");
      for (const imp of mapped.imports) imports.add(imp);
      if (mapped.go === "time.Time" || mapped.go === "*time.Time") imports.add("time");

      const parts = [`column:${col.column_name}`];
      if (mapped.tagType) parts.push(`type:${mapped.tagType}`);
      if (pks.includes(col.column_name)) parts.push("primaryKey");

      const fieldGo = snakeToPascal(col.column_name);
      fields.push(`\t${fieldGo} ${mapped.go} \`gorm:"${parts.join(";")}"\``);
    }

    const goImports = [...imports].sort();
    const importBlock = goImports.length
      ? `import (\n${goImports.map((i) => `\t"${i}"`).join("\n")}\n)\n\n`
      : "";

    const pkNote = pks.length > 1
      ? `//\n// 复合主键（${pks.join(" + ")}）：GORM 多 primaryKey 字段即联合主键；\n// 手写层 upsert/delete 按两列定位（link=整行覆盖，unlink=幂等 no-op）。\n`
      : "";

    const content = `// Code generated by scripts/scaffold-models.mjs. DO NOT EDIT.
// DB-First scaffold：${table}（家族 DB-First 口径；schema SSOT = saas shared drizzle
// public.${table}，本仓禁 migrate/seed）。由 information_schema 反推生成——
// 禁手改，L4.codegen.idempotent strict 复跑字节比对会暴露手改。手写 repository/
// 常量/领域方法写在 internal/impl/ 层。
//
// 字段含义见 saas-identity-platform-shared/src/db/schema.ts。
${pkNote}
package entity

${importBlock}\
// ${structName} maps public.${table}.
type ${structName} struct {
${fields.join("\n")}
}

// TableName GORM 表名（默认蛇形复数会错，显式钉死）。
func (${structName}) TableName() string { return "${table}" }
`;

    writeFileSync(resolve(OUTPUT_DIR, `${table}.go`), content.replace(/\r/g, ""), "utf8");
    console.log(`[scaffold-models]   ${table}.go（${columns.length} 列，PK=${pks.join("+") || "无"}）`);
  }

  await client.end();

  // 3. gofmt 规范生成区（laravel pint 同款——保证 L1 门干净 + 字节幂等前提）
  execSync(`gofmt -w "${OUTPUT_DIR}"`, { stdio: "inherit" });

  // 4. 漂移护栏（tracked diff ∪ untracked）
  const dirty = (() => {
    try {
      execSync(`git diff --exit-code --quiet -- "${ZONE}"`, { cwd: ROOT, stdio: "ignore" });
      return false;
    } catch {
      return true;
    }
  })();
  const untracked = execSync(
    `git ls-files --others --exclude-standard -- "${ZONE}"`,
    { cwd: ROOT, encoding: "utf8" },
  ).trim();
  if (dirty || untracked) {
    console.error("[scaffold-models] FATAL: 生成区与 git HEAD 不一致（含 untracked 新文件）");
    console.error("[scaffold-models]        处理：确认库 schema 变更合法（shared 已 db:migrate 且连的是本族库）后");
    console.error(`[scaffold-models]        git add ${ZONE} && git commit，再重跑本脚本核绿`);
    process.exit(2);
  }
  console.log("[scaffold-models] OK — 生成区与 HEAD 一致");
}

mkdirSync(OUTPUT_DIR, { recursive: true });
main().catch((e) => {
  console.error("[scaffold-models] FATAL:", e.message);
  process.exit(2);
});
