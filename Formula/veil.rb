# typed: false
# frozen_string_literal: true

# Homebrew formula for veil-cli.
#
# Install with:
#   brew tap thatsbass/veil-cli
#   brew install veil
#
class Veil < Formula
  desc "CLI client for the Veil LLM gateway"
  homepage "https://github.com/thatsbass/veil-cli"
  url "https://github.com/thatsbass/veil-cli/archive/refs/tags/v1.0.0.tar.gz"
  sha256 "" # replace with: curl -fsSL <url> | sha256sum
  license "MIT"
  version "1.0.0"

  depends_on "go" => :build

  def install
    system "go", "build", "-ldflags=-w -s", "-o", bin/"veil", "cmd/main.go"
  end

  test do
    assert_match "veil", shell_output("#{bin}/veil version")
  end
end
