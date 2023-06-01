# typed: strict
# frozen_string_literal: true

class User < ApplicationRecord
  extend Enumerize

  enumerize :locale, in: I18n.available_locales

  has_many :profile_members, dependent: :restrict_with_exception
  has_many :profiles, dependent: :restrict_with_exception, through: :profile_members

  has_secure_password

  validates :email, presence: true, uniqueness: true

  sig { returns(Profile) }
  def first_profile
    T.let(profiles.first, Profile)
  end

  sig { returns(T::Boolean) }
  def track_sign_in
    old_current, new_current = current_signed_in_at, Time.current

    self.last_signed_in_at = old_current || new_current
    self.current_signed_in_at = new_current
    self.sign_in_count += 1

    save!(validate: false)
  end
end
