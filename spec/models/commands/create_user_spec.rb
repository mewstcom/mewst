# typed: false
# frozen_string_literal: true

RSpec.describe Commands::CreateUser do
  context "when valid" do
    let!(:phone_number) { "+819000000000" }
    let!(:phone_number_verification_challenge) { create(:phone_number_verification_challenge, phone_number:) }
    let!(:idname) { "hello" }
    let!(:locale) { "en" }
    let!(:command) { Commands::CreateUser.new(phone_number_verification_challenge:, idname:, locale:) }

    it "creates an account" do
      expect(PhoneNumberVerificationChallenge.count).to eq(1)
      expect(PhoneNumber.count).to eq(0)
      expect(Profile.count).to eq(0)
      expect(UserPhoneNumber.count).to eq(0)
      expect(UserProfile.count).to eq(0)
      expect(User.count).to eq(0)

      command.call

      expect(PhoneNumberVerificationChallenge.count).to eq(0)
      expect(PhoneNumber.count).to eq(1)
      expect(Profile.count).to eq(1)
      expect(UserPhoneNumber.count).to eq(1)
      expect(UserProfile.count).to eq(1)
      expect(User.count).to eq(1)

      user = command.user

      expect(user).to have_attributes(
        sign_in_count: 0,
        current_signed_in_at: nil,
        last_signed_in_at: nil
      )

      expect(user.phone_number).to have_attributes(
        value: phone_number
      )

      expect(user.profile).to have_attributes(
        profilable_type: "user",
        idname:,
        locale:,
        name: "",
        description: "",
        deleted_at: nil
      )
    end
  end
end
