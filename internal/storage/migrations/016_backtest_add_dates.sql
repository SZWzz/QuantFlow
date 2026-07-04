-- 016_backtest_add_dates: add backtest period date columns

ALTER TABLE backtest_results ADD COLUMN backtest_start TEXT NOT NULL DEFAULT '';
ALTER TABLE backtest_results ADD COLUMN backtest_end TEXT NOT NULL DEFAULT '';
