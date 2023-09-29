# typed: false
# frozen_string_literal: true

RSpec.describe Latest::PostForm do
  let!(:viewer) { create(:actor) }

  context "when invalid" do
    context "when comment is blank" do
      let!(:form) { Latest::PostForm.new(viewer:, comment: "") }

      it "is invalid" do
        expect(form.invalid?).to be(true)
        expect(form.errors.where(:comment, :blank)).to be_present
      end
    end

    context "when comment length is too long" do
      let!(:long_comment) { "a" * (Post::MAXIMUM_COMMENT_LENGTH + 1) }
      let!(:form) { Latest::PostForm.new(viewer:, comment: long_comment) }

      it "is invalid" do
        expect(form.invalid?).to be(true)
        expect(form.errors.where(:comment, :too_long)).to be_present
      end
    end
  end
end
