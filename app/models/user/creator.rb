# typed: strict
# frozen_string_literal: true

class User::Creator
  extend T::Sig

  include ActiveModel::Model

  sig { returns(T.nilable(User)) }
  attr_reader :user

  validates :idname, format: {with: Profile::IDNAME_FORMAT}, length: {maximum: 20}, presence: true
  validate :idname_uniqueness

  sig { params(phone_number_verification: PhoneNumberVerification, idname: T.nilable(String)).void }
  def initialize(phone_number_verification:, idname: nil)
    @phone_number_verification = phone_number_verification
    @idname = idname
    @user = T.let(nil, T.nilable(User))
  end

  sig { returns(T.self_type) }
  def call
    ActiveRecord::Base.transaction do
      @user = User.create!
      phone_number = PhoneNumber.find_or_create_by!(value: phone_number_verification.phone_number)

      @user.create_user_phone_number!(phone_number:)

      profile = @user.create_profile!(idname:, profilable_type: :user)
      @user.create_user_profile!(profile:)

      phone_number_verification.destroy
    end

    self
  end

  private

  sig { returns(PhoneNumberVerification) }
  attr_reader :phone_number_verification

  sig { returns(T.nilable(String)) }
  attr_reader :idname

  sig { returns(T.untyped) }
  def idname_uniqueness
    if Profile.find_by(idname:)
      errors.add(:idname, :idname_uniqueness)
    end
  end
end
