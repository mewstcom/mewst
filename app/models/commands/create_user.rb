# typed: strict
# frozen_string_literal: true

class Commands::CreateUser < Commands::Base
  include ActiveModel::Model

  sig { returns(T.nilable(User)) }
  attr_reader :user

  delegate :phone_number, to: :phone_number_verification_challenge

  validates :idname, format: {with: Profile::IDNAME_FORMAT}, length: {maximum: 20}, presence: true
  validates :locale, inclusion: I18n.available_locales.map(&:to_s)
  validate :idname_uniqueness

  sig { params(phone_number_verification_challenge: PhoneNumberVerificationChallenge, idname: T.nilable(String), locale: T.nilable(String)).void }
  def initialize(phone_number_verification_challenge:, idname: nil, locale: nil)
    @phone_number_verification_challenge = phone_number_verification_challenge
    @idname = idname
    @locale = locale
    @user = T.let(nil, T.nilable(User))
  end

  sig { returns(T.self_type) }
  def call
    ActiveRecord::Base.transaction do
      @user = User.create!
      phone_number = PhoneNumber.find_or_create_by!(value: phone_number_verification_challenge.phone_number)

      @user.create_user_phone_number!(phone_number:)

      profile = @user.create_profile!(idname:, profilable_type: :user, locale:)
      @user.create_user_profile!(profile:)

      phone_number_verification_challenge.destroy
    end

    self
  end

  sig { returns(User) }
  def user!
    T.cast(user, User)
  end

  private

  sig { returns(PhoneNumberVerificationChallenge) }
  attr_reader :phone_number_verification_challenge

  sig { returns(T.nilable(String)) }
  attr_reader :idname

  sig { returns(T.nilable(String)) }
  attr_reader :locale

  sig { returns(T.untyped) }
  def idname_uniqueness
    if Profile.find_by(idname:)
      errors.add(:idname, :idname_uniqueness)
    end
  end
end
