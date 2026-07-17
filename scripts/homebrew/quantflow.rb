# build/homebrew/quantflow.rb
# Homebrew formula for QuantFlow Terminal
#
# Usage:
#   brew tap SZWzz/homebrew-tap
#   brew install quantflow
#
# SHA256 placeholders must be replaced with actual values from the release.

class Quantflow < Formula
  desc "Dual-mode quantitative finance terminal (Bloomberg-style + workflow orchestration)"
  homepage "https://github.com/SZWzz/QuantFlow"
  license "AGPL-3.0"
  version "2026.7.17"

  if Hardware::CPU.arm?
    url "https://github.com/SZWzz/QuantFlow/releases/download/v#{version}/quantflow-#{version}-darwin-arm64.dmg"
    sha256 "0000000000000000000000000000000000000000000000000000000000000000" # REPLACE with actual
  else
    url "https://github.com/SZWzz/QuantFlow/releases/download/v#{version}/quantflow-#{version}-darwin-amd64.dmg"
    sha256 "0000000000000000000000000000000000000000000000000000000000000000" # REPLACE with actual
  end

  depends_on "go" => :optional

  def install
    app_path = "QuantFlow.app"
    prefix.install app_path
    bin.write_exec_script "#{prefix}/#{app_path}/Contents/MacOS/quantflow"
  end

  def caveats
    <<~EOS
      QuantFlow Terminal installed to #{prefix}/QuantFlow.app.

      Run with:
        open -a QuantFlow
      Or from command line:
        quantflow

      Python sidecar (optional, for ML/AI features):
        cd #{prefix}/QuantFlow.app/Contents/Resources/python
        python3 -m venv venv
        source venv/bin/activate
        pip install -r requirements.txt
        python -m src.server
    EOS
  end

  test do
    system "#{bin}/quantflow", "--version"
  end
end
