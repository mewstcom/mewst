# typed: false
# frozen_string_literal: true

RSpec.describe Url do
  describe "#valid?" do
    context "URLが不正なとき" do
      it "`false` を返す" do
        # ただの文字列
        expect(Url.new("invalid_url").valid?).to eq(false)
        # スペースが含まれている
        expect(Url.new("https: //annict.com").valid?).to eq(false)
        # ドメインが含まれていない
        expect(Url.new("https://").valid?).to eq(false)
        # プロトコルが含まれていない
        expect(Url.new("annict.com").valid?).to eq(false)
      end
    end

    context "URLが正しいとき" do
      it "`true` を返す" do
        expect(Url.new("https://annict.com").valid?).to eq(true)
      end
    end
  end

  describe "#host_and_path" do
    context "URLが不正なとき" do
      it "`nil` を返すこと" do
        expect(Url.new("invalid_url").host_and_path).to be_nil
      end
    end

    context "URLが正しいとき" do
      it "ホストとパスを返すこと" do
        expect(Url.new("https://example.com/path").host_and_path).to eq("example.com/path")
      end
    end
  end
end
