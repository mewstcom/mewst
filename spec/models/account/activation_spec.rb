# typed: false
# frozen_string_literal: true

RSpec.describe Account::Activation do
  context "when valid" do
    let!(:phone_number) { "+819000000000" }
    let!(:phone_number_verification) { create(:phone_number_verification, phone_number:) }
    let!(:atname) { "shimbaco" }
    let!(:locale) { "en" }
    let!(:activation) { Account::Activation.new(atname:, locale:) }

    before do
      activation.phone_number_verification = phone_number_verification
    end

    it "creates an account" do
      expect(PhoneNumberVerification.count).to eq(1)
      expect(Profile.count).to eq(0)
      expect(AccountProfile.count).to eq(0)
      expect(Account.count).to eq(0)

      account = activation.run

      expect(PhoneNumberVerification.count).to eq(0)
      expect(Profile.count).to eq(1)
      expect(AccountProfile.count).to eq(1)
      expect(Account.count).to eq(1)

      expect(account).to have_attributes(
        phone_number:,
        sign_in_count: 0,
        current_signed_in_at: nil,
        last_signed_in_at: nil
      )

      expect(account.profile).to have_attributes(
        profilable_type: "account",
        atname:,
        locale:,
        name: "",
        description: "",
        deleted_at: nil
      )
    end
  end
end
