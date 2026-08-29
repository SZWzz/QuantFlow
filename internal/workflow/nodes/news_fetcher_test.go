package nodes

import (
	"context"
	"quantflow/internal/market/adapters"
	"testing"
)

func TestNewsFetcherNode_Interface(t *testing.T) {
	node, err := NewNewsFetcherNode("news-1", map[string]any{"source": "eastmoney"})
	if err != nil {
		t.Fatalf("NewNewsFetcherNode: %v", err)
	}
	if node.ID() != "news-1" {
		t.Errorf("expected id 'news-1', got %s", node.ID())
	}
	if node.NodeType() != "news_fetcher" {
		t.Errorf("expected node_type 'news_fetcher', got %s", node.NodeType())
	}
	if node.Category() != "research" {
		t.Errorf("expected category 'research', got %s", node.Category())
	}
}

func TestNewsFetcherNode_Ports(t *testing.T) {
	node, _ := NewNewsFetcherNode("news-1", nil)

	inputs := node.InputPorts()
	if len(inputs) != 2 {
		t.Errorf("expected 2 input ports, got %d", len(inputs))
	}
	if inputs[0].Name != "symbol" || !inputs[0].Required {
		t.Error("first input must be 'symbol' and required")
	}

	outputs := node.OutputPorts()
	if len(outputs) != 2 {
		t.Errorf("expected 2 output ports, got %d", len(outputs))
	}
	if outputs[0].Name != "news_text" {
		t.Errorf("expected 'news_text' output, got %s", outputs[0].Name)
	}
}

func TestNewsFetcherNode_Execute_Mock(t *testing.T) {
	node, _ := NewNewsFetcherNode("news-1", map[string]any{})

	result, err := node.Execute(context.Background(),
		map[string]any{"symbol": "600519"},
		map[string]any{"source": "eastmoney", "language": "zh"}, nil,
	)
	if err != nil {
		t.Fatalf("Execute should not error in mock mode: %v", err)
	}

	newsText, ok := result["news_text"].(string)
	if !ok {
		t.Fatal("news_text output must be a string")
	}
	if len(newsText) == 0 {
		t.Error("news_text should not be empty (mock should provide fallback)")
	}
	t.Logf("news_text: %s", newsText)

	articles, ok := result["articles"].([]adapters.NewsArticle)
	if !ok {
		t.Fatal("articles output must be []adapters.NewsArticle")
	}
	if len(articles) == 0 {
		t.Error("articles should contain at least mock data")
	}
}

func TestNewsFetcherNode_Execute_MissingSymbol(t *testing.T) {
	node, _ := NewNewsFetcherNode("news-1", nil)
	_, err := node.Execute(context.Background(), map[string]any{}, nil, nil)
	if err == nil {
		t.Error("expected error for missing symbol")
	}
}

func TestMockNewsArticles(t *testing.T) {
	articles := mockNewsArticles("600519", 3)
	if len(articles) != 1 {
		t.Errorf("mock should always return 1 article, got %d", len(articles))
	}
	if articles[0].Symbol != "600519" {
		t.Errorf("expected symbol 600519, got %s", articles[0].Symbol)
	}
}
