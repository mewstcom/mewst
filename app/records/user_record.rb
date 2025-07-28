# typed: strict
# frozen_string_literal: true

class UserRecord < ApplicationRecord
  self.table_name = "users"

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
end