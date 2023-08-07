# typed: false
# frozen_string_literal: true

RSpec.describe Forms::Repost do
  let!(:profile) { create(:profile, :for_user) }

  context "when invalid" do
    context "when post is blank" do
      let!(:form) { Forms::Repost.new(viewer: profile, target_post_id: "unknown") }

      it "is invalid" do
        expect(form.invalid?).to be(true)
        expect(form.errors.where(:target_post, :blank)).to be_present
      end
    end
  end
end
