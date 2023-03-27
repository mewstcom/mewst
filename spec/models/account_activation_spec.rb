# typed: false
# frozen_string_literal: true

RSpec.describe AccountActivation do
  context "when valid" do
    let!(:email) { "test@example.com" }
    let!(:verification) { create(:verification, email:) }
    let!(:atname) { "shimbaco" }
    let!(:locale) { "en" }
    let!(:password) { "shimbaco" }
    let!(:activation) { AccountActivation.new(atname:, locale:, password:) }

    before do
      activation.verification = verification
    end

    it "creates an account" do
      expect(Verification.count).to eq(1)
      expect(Profile.count).to eq(0)
      expect(ProfileMember.count).to eq(0)
      expect(Account.count).to eq(0)

      account = activation.run

      expect(Verification.count).to eq(0)
      expect(Profile.count).to eq(1)
      expect(ProfileMember.count).to eq(1)
      expect(Account.count).to eq(1)

      expect(account).to have_attributes(
        email:,
        locale:,
        sign_in_count: 0,
        current_signed_in_at: nil,
        last_signed_in_at: nil
      )

      expect(account.first_profile).to have_attributes(
        atname:,
        name: "",
        description: "",
        deleted_at: nil
      )
    end
  end
end
