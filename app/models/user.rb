# typed: strict
# frozen_string_literal: true

class User < ApplicationRecord
  has_one :user_phone_number, dependent: :restrict_with_exception
  has_one :user_profile, dependent: :restrict_with_exception
  has_one :phone_number, dependent: :restrict_with_exception, through: :user_phone_number
  has_one :profile, dependent: :restrict_with_exception, through: :user_profile

  sig { returns(T::Boolean) }
  def track_sign_in
    old_current, new_current = current_signed_in_at, Time.current

    self.last_signed_in_at = old_current || new_current
    self.current_signed_in_at = new_current
    self.sign_in_count += 1

    save!(validate: false)
  end
end
