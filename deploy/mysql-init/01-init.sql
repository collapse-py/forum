# MySQL 容器第一次啟動時執行的初始化腳本。
#
# 【這個目錄的作用】
# mysql:8.0 的映像會在**資料目錄是空的**那次啟動，執行
# /docker-entrypoint-initdb.d/ 底下的所有檔案。因此這個目錄的內容只會被執行一次 ——
# 而「只會被執行一次」正是它最大的陷阱：改了這個檔之後 `docker compose down -v`
# 不會重跑（因為 -v 會刪掉資料目錄，那確實會重跑），但 `docker compose restart`
# 永遠不會。
#
# 因此這個目錄裡的腳本必須是**冪等**的 —— 也就是說，跑第二次不會出錯。
# 這是唯一的例外：建立資料庫（CREATE DATABASE）本來就是冪等的。

# 為什麼需要這個檔：MYSQL_DATABASE 環境變數已經會建立資料庫，而 DSN 裡的
# 庫名通常與它一致。這個檔的存在是為了**設定字元集** ——
# MYSQL_DATABASE 建立的庫會用 mysql:8.0 的預設（utf8mb4_0900_ai_ci，在 8.0 上
# 已經是預設），但明確寫一次可以讓「這個庫的字元集是刻意選的」變成一個
# 可以 grep 出來的事實。
#
# 挑 utf8mb4 而非 utf8mb3：utf8mb3（俗稱 utf8）無法存 emoji 與部分罕用漢字。
# 症狀是 "Incorrect string value: '\xF0\x9F...'"，而那個訊息不會指出真正的
# 原因是字元集。
#
# 挑 utf8mb4_unicode_ci 而非 utf8mb4_0900_ai_ci：兩者都支援中文，
# 但 unicode_ci 的排序規則在 MySQL 5.7 與 8.0 之間一致 —— 若日後有人把資料
# 匯出再匯到別的版本，排序順序不會變。0900_ai_ci 是 8.0 專屬的。
CREATE DATABASE IF NOT EXISTS `forum`
    CHARACTER SET utf8mb4
    COLLATE utf8mb4_unicode_ci;

-- 一個「這個目錄被執行了」的證據。
--
-- 為什麼需要它：這個目錄只會執行一次，而上面那個 CREATE DATABASE 是冪等的，
-- 因此沒有任何可觀察的痕跡。當有人在半年後問「這個腳本到底跑過沒有」時，
-- 這張表就是答案。
--
-- 刻意不寫進任何資料：這是一個「檔案存在」的標記，不是資料。
-- 若寫入時間戳，它在每次 dump 時都會不同而製造無意義的 diff。
CREATE TABLE IF NOT EXISTS `_init_marker` (
    `name`        VARCHAR(64)  NOT NULL PRIMARY KEY,
    `note`        VARCHAR(255) NOT NULL
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

INSERT IGNORE INTO `_init_marker` (`name`, `note`) VALUES (
    'deploy/mysql-init',
    '這個資料庫的字元集是 utf8mb4_unicode_ci；schema 由後端啟動時自動遷移，不需要手動執行任何 SQL'
);

-- 刻意**不**在這裡建立資料表。
--
-- 理由：這個專案的 schema 由後端啟動時的 data.MigrateMySQL 建立（見
-- backend/main.go 的啟動順序）。這是刻意的單一來源原則：schema 的定義只在
-- Go 程式碼裡有一份，而它在應用程式碼與資料庫之間保持同步。
--
-- 在這個目錄裡另外寫一份 CREATE TABLE 會立刻製造第二個來源，而兩份不一致時的
-- 症狀是「某個欄位在開發環境有、在正式環境沒有」，而那不會出現在任何錯誤裡 ——
-- 直到某個查詢在那個環境上失敗。
--
-- 因此這個目錄的職責只有一個：確保資料庫存在，且它的字元集是對的。