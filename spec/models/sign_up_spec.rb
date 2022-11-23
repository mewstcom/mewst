# typed: false
# frozen_string_literal: true

RSpec.describe SignUp, type: :model do
  describe "#create" do
    let!(:email) { "hello@example.com" }
    let!(:idname) { "hello" }

    it "creates an account" do
      expect(Account.count).to eq(0)
      expect(User.count).to eq(0)
      expect(Profile.count).to eq(0)

      sign_up = SignUp.new(email:, idname:)
      sign_up.create

      expect(Account.count).to eq(1)
      expect(User.count).to eq(1)
      expect(Profile.count).to eq(1)
      account, user = Account.first, User.first

      expect(account).to have_attributes(
        email:,
        sign_in_count: 0,
        current_signed_in_at: nil,
        last_signed_in_at: nil,
      )

      expect(user).to have_attributes(
        account_id: account.id,
      )

      expect(Profile.first).to have_attributes(
        profilable_type: "User",
        profilable_id: user.id,
        idname:,
        name: "",
        description: "",
        deleted_at: nil,
      )
    end
  end
end
