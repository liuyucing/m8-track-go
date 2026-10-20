-- 2026-10-20 历史数据修复：track_sync_detail 重复轨迹清理 + 已签收订单地点回填
--
-- 背景：去重键曾把事件时间转 UTC 比较，而 go-mssqldb 写 DATETIME 列时丢弃时区偏移
-- 只留墙上时钟，读写两侧键不一致，导致每次同步把整串轨迹全量重复入库
-- （修复见 internal/service/trackevents.go eventDedupKey 注释）。
-- 该脚本已于 2026-10-20 对生产库执行完毕（1,221,559 行 → 27,201 行），仅作留档；
-- 若再次出现重复（理论上修复后不会）可重复执行，幂等。
--
-- 1) 轨迹明细去重：按 (md_no, 墙上时钟分钟, stage, event_desc) 保留每组最早一条
WITH g AS (
    SELECT id, ROW_NUMBER() OVER (
        PARTITION BY md_no, CONVERT(varchar(16), event_time, 120), ISNULL(stage,''), ISNULL(event_desc,'')
        ORDER BY id) AS rn
    FROM track_sync_detail
)
DELETE FROM g WHERE rn > 1;

-- 2) 已签收订单回填 scBNDtl.location / locationmatch
--    （新代码在签收时自动回写；本脚本只补历史订单。匹配规则与 service.MatchLocation 一致：
--     忽略大小写/分隔符的整串包含，否则按逗号分段（剔除两位国家代码）全部命中。）
-- 执行方式：由 tools 临时程序按 service.MatchLocation 逐行计算后 UPDATE，未用纯 SQL 实现。
-- 如需手工核对：
SELECT d.FID, LTRIM(RTRIM(d.MDNo)) AS md_no, t.location, d.CustShipInfo, d.location, d.locationmatch
FROM scBNDtl d
OUTER APPLY (
    SELECT TOP 1 td.location FROM track_sync_detail td
    WHERE td.md_no = LTRIM(RTRIM(d.MDNo)) AND td.location IS NOT NULL AND LTRIM(RTRIM(td.location)) <> ''
    ORDER BY td.event_time DESC, td.id DESC
) t
WHERE d.TrackDelivered = 1 AND (d.location IS NULL OR LTRIM(RTRIM(d.location)) = '');
