# typed: false
# frozen_string_literal: true

RSpec.describe Latest::PostForm do
  let!(:viewer) { create(:actor) }

  context "入力データが不正なとき" do
    context "投稿内容が空白のとき" do
      let!(:form) { Latest::PostForm.new(viewer:, comment: "") }

      it "不正なデータとすること" do
        expect(form.invalid?).to be(true)
        expect(form.errors.where(:comment, :blank)).to be_present
      end
    end

    context "投稿内容が長すぎるとき" do
      let!(:long_comment) { "a" * (Post::MAXIMUM_COMMENT_LENGTH + 1) }
      let!(:form) { Latest::PostForm.new(viewer:, comment: long_comment) }

      it "不正なデータとすること" do
        expect(form.invalid?).to be(true)
        expect(form.errors.where(:comment, :too_long)).to be_present
      end
    end
  end
end
