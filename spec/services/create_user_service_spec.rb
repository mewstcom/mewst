# typed: false
# frozen_string_literal: true

RSpec.describe CreateUserService do
  context "when valid" do
    let!(:phone_number) { "+819000000000" }
    let!(:phone_number_confirmation) { create(:phone_number_confirmation, phone_number:) }
    let!(:idname) { "hello" }
    let!(:form) { NewUserForm.new(phone_number_confirmation:, idname:) }

    it "creates an account" do
      expect(PhoneNumberConfirmation.count).to eq(1)
      expect(PhoneNumber.count).to eq(0)
      expect(Profile.count).to eq(0)
      expect(UserPhoneNumber.count).to eq(0)
      expect(UserProfile.count).to eq(0)
      expect(User.count).to eq(0)

      result = CreateUserService.new(form:).call

      expect(PhoneNumberConfirmation.count).to eq(0)
      expect(PhoneNumber.count).to eq(1)
      expect(Profile.count).to eq(1)
      expect(UserPhoneNumber.count).to eq(1)
      expect(UserProfile.count).to eq(1)
      expect(User.count).to eq(1)

      user = result.user

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
        name: "",
        description: "",
        deleted_at: nil
      )
    end
  end
end
