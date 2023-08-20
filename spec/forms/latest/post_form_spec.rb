# typed: false
# frozen_string_literal: true

RSpec.describe Latest::PostForm do
  let!(:user) { create(:user) }
  let!(:profile) { user.profile }

  context "when invalid" do
    context "when comment is blank" do
      let!(:form) { Latest::PostForm.new(profile:, comment: "") }

      it "is invalid" do
        expect(form.invalid?).to be(true)
        expect(form.errors.where(:comment, :blank)).to be_present
      end
    end

    context "when comment length is too long" do
      let!(:long_comment) { "a" * (Post::MAXIMUM_COMMENT_LENGTH + 1) }
      let!(:form) { Latest::PostForm.new(profile:, comment: long_comment) }

      it "is invalid" do
        expect(form.invalid?).to be(true)
        expect(form.errors.where(:comment, :too_long)).to be_present
      end
    end
  end
end
