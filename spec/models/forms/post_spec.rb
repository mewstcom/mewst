# typed: false
# frozen_string_literal: true

RSpec.describe Forms::Post do
  let!(:profile) { create(:profile, :with_account) }

  context "when invalid" do
    context "when already reposted" do
      let!(:form) { Forms::Repost.new(profile:, post_id) }
      before do
      end

      it do
        binding.irb
      end
    end
  end
end
