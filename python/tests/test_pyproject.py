import tomllib


def test_pyproject_has_build_system():
    with open("pyproject.toml", "rb") as f:
        data = tomllib.load(f)
    assert "build-system" in data, "Missing [build-system] table"
    requires = data["build-system"]["requires"]
    # grpcio-tools is a build-time requirement (proto codegen), not a runtime dep
    assert any(r.startswith("setuptools>=") for r in requires)
    assert any(r.startswith("grpcio-tools>=") for r in requires)


def test_mootdx_in_data_not_dev():
    with open("pyproject.toml", "rb") as f:
        data = tomllib.load(f)
    data_deps = data["project"]["optional-dependencies"].get("data", [])
    dev_deps = data["project"]["optional-dependencies"].get("dev", [])
    assert any("mootdx" in dep for dep in data_deps), "mootdx should be in data deps"
    assert not any("mootdx" in dep for dep in dev_deps), "mootdx should NOT be in dev deps"


def test_torch_in_ml_optional():
    with open("pyproject.toml", "rb") as f:
        data = tomllib.load(f)
    ml_deps = data["project"]["optional-dependencies"].get("ml", [])
    assert any("torch" in dep for dep in ml_deps), "torch should be in ml deps"
