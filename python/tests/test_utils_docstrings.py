import inspect


def test_validate_dates_has_docstring():
    from src.data.utils import validate_dates
    doc = validate_dates.__doc__
    assert doc is not None and len(doc) > 30, "validate_dates needs a proper docstring"


def test_get_1m_bars_has_docstring():
    from src.data.utils import get_1m_bars
    doc = get_1m_bars.__doc__
    assert doc is not None and len(doc) > 30, "get_1m_bars needs a proper docstring"


def test_validate_dates_describes_args():
    from src.data.utils import validate_dates
    doc = validate_dates.__doc__
    assert "start" in doc.lower() or "end" in doc or "date" in doc.lower()


def test_get_1m_bars_describes_return():
    from src.data.utils import get_1m_bars
    doc = get_1m_bars.__doc__
    assert "return" in doc.lower() or "yield" in doc.lower() or "generator" in doc.lower()
