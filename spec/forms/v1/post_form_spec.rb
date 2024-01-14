# typed: false
# frozen_string_literal: true

RSpec.describe V1::PostForm do
  let!(:viewer) { create(:actor) }

  context "入力データが不正なとき" do
    context "投稿内容が空白のとき" do
      let!(:form) { V1::PostForm.new(viewer:, content: "") }

      it "不正なデータとすること" do
        expect(form.invalid?).to be(true)
        expect(form.errors.where(:content, :blank)).to be_present
      end
    end

    context "投稿内容が長すぎるとき" do
      let!(:long_content) { "a" * (Post::MAXIMUM_CONTENT_LENGTH + 1) }
      let!(:form) { V1::PostForm.new(viewer:, content: long_content) }

      it "不正なデータとすること" do
        expect(form.invalid?).to be(true)
        expect(form.errors.where(:content, :too_long)).to be_present
      end
    end
  end
end
