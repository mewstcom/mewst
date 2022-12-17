# typed: false
# frozen_string_literal: true

RSpec.describe User::Creator do
  context "when valid" do
    let!(:phone_number) { "+819000000000" }
    let!(:phone_number_verification) { create(:phone_number_verification, phone_number:) }
    let!(:idname) { "hello" }
    let!(:locale) { "en" }
    let!(:user_creator) { User::Creator.new(phone_number_verification:, idname:, locale:) }

    it "creates an account" do
      expect(PhoneNumberVerification.count).to eq(1)
      expect(PhoneNumber.count).to eq(0)
      expect(Profile.count).to eq(0)
      expect(UserPhoneNumber.count).to eq(0)
      expect(UserProfile.count).to eq(0)
      expect(User.count).to eq(0)

      user_creator.call

      expect(PhoneNumberVerification.count).to eq(0)
      expect(PhoneNumber.count).to eq(1)
      expect(Profile.count).to eq(1)
      expect(UserPhoneNumber.count).to eq(1)
      expect(UserProfile.count).to eq(1)
      expect(User.count).to eq(1)

      user = user_creator.user

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
