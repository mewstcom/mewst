# typed: strict
# frozen_string_literal: true

class User < ApplicationRecord
  extend Enumerize

  PASSWORD_MIN_LENGTH = 8

  enumerize :locale, in: I18n.available_locales

  has_many :actors, dependent: :restrict_with_exception
  has_one :user_profile, dependent: :restrict_with_exception
  has_one :profile, through: :user_profile

  has_secure_password

  validates :email, presence: true, uniqueness: true

  sig { returns(Actor) }
  def first_actor
    actors.find_by!(profile:)
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
