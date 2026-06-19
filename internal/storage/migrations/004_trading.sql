-- 004_trading: orders, trades, and positions tables for trading engine

CREATE TABLE IF NOT EXISTS orders (
    id TEXT PRIMARY KEY,
    symbol TEXT NOT NULL,
    side TEXT NOT NULL CHECK(side IN ('buy', 'sell')),
    order_type TEXT NOT NULL CHECK(order_type IN ('market', 'limit', 'stop')),
    quantity REAL NOT NULL CHECK(quantity > 0),
    price REAL,
    stop_price REAL,
    filled_qty REAL NOT NULL DEFAULT 0,
    filled_avg_price REAL,
    status TEXT NOT NULL DEFAULT 'pending' CHECK(status IN ('pending', 'partial', 'filled', 'cancelled', 'rejected')),
    placed_at INTEGER NOT NULL,
    filled_at INTEGER
);

CREATE INDEX IF NOT EXISTS idx_orders_symbol ON orders(symbol);
CREATE INDEX IF NOT EXISTS idx_orders_status ON orders(status);

CREATE TABLE IF NOT EXISTS trades (
    id TEXT PRIMARY KEY,
    order_id TEXT NOT NULL REFERENCES orders(id),
    symbol TEXT NOT NULL,
    side TEXT NOT NULL,
    quantity REAL NOT NULL,
    price REAL NOT NULL,
    trade_at INTEGER NOT NULL
);

CREATE INDEX IF NOT EXISTS idx_trades_order ON trades(order_id);
CREATE INDEX IF NOT EXISTS idx_trades_symbol ON trades(symbol, trade_at);

CREATE TABLE IF NOT EXISTS positions (
    symbol TEXT PRIMARY KEY,
    quantity REAL NOT NULL,
    avg_price REAL NOT NULL,
    updated_at INTEGER NOT NULL
);
