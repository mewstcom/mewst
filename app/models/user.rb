# typed: strict
# frozen_string_literal: true

class User < ApplicationRecord
  extend Enumerize

  enumerize :locale, in: I18n.available_locales

  belongs_to :profile
  has_many :actors, dependent: :restrict_with_exception

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
