# typed: false
# frozen_string_literal: true

RSpec.describe Forms::Repost do
  let!(:profile) { create(:profile, :with_account) }

  context "when invalid" do
    context "when post is blank" do
      let!(:form) { Forms::Repost.new(profile:, post_id: "unknown") }

      it "should be invalid" do
        expect(form.invalid?).to be(true)
        expect(form.errors.where(:post, :blank)).to be_present
      end
    end
  end
end
