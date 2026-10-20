-- !!! 手动执行：ERP 共享表 scBNDtl，请在 DBA 维护窗口运行，并先确认列名/权限。
-- 幂等：为 scBNDtl 增加发货/签收日期回写列（DATETIME NULL）。
IF COL_LENGTH('dbo.scBNDtl','FCshipDate') IS NULL
    ALTER TABLE dbo.scBNDtl ADD FCshipDate DATETIME NULL;
IF COL_LENGTH('dbo.scBNDtl','FCdeliveredDate') IS NULL
    ALTER TABLE dbo.scBNDtl ADD FCdeliveredDate DATETIME NULL;
