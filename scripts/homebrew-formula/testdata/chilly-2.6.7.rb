# typed: false
# frozen_string_literal: true

class Chilly < Formula
  desc "Search chill.institute and send transfers from the terminal"
  homepage "https://chill.institute"
  license "MIT"

  on_macos do
    on_intel do
      url "https://github.com/chill-institute/chill-cli/releases/download/v2.6.7/chilly_2.6.7_darwin_amd64.tar.gz"
      sha256 "952caf0f17550a825e59e57832e63eb921ce988b1168bcd35e69d6e483ddbae2"
    end
    on_arm do
      url "https://github.com/chill-institute/chill-cli/releases/download/v2.6.7/chilly_2.6.7_darwin_arm64.tar.gz"
      sha256 "88c5ab7e8b46f0a1ed9692d78917c65c9d73760d45708eed1ab97499b7ed7fce"
    end
  end

  on_linux do
    on_intel do
      url "https://github.com/chill-institute/chill-cli/releases/download/v2.6.7/chilly_2.6.7_linux_amd64.tar.gz"
      sha256 "ec029aebf4d3197c50d302fd754148a8b81a4bf752937029eca2fbf7f78a3bed"
    end
    on_arm do
      url "https://github.com/chill-institute/chill-cli/releases/download/v2.6.7/chilly_2.6.7_linux_arm64.tar.gz"
      sha256 "033ca984f08da58b686092895ff090d2cd91d3b0c04e23feb33baced7b481f84"
    end
  end

  def install
    bin.install "chilly"
  end

  test do
    system bin/"chilly", "version", "--output", "json"
  end
end
