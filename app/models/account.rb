# typed: strict
# frozen_string_literal: true

class Account < ApplicationRecord
  has_many :email_confirmations
  has_many :users

  validates :email, email: true, presence: true, uniqueness: true

  sig { returns(User) }
  def default_user
    users.order(:id).first
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
