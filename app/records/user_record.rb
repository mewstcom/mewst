# typed: strict
# frozen_string_literal: true

class UserRecord < ApplicationRecord
  self.table_name = "users"

  extend Enumerize

  PASSWORD_MIN_LENGTH = 8

  enumerize :locale, in: I18n.available_locales

  has_many :actor_records, class_name: "ActorRecord", dependent: :restrict_with_exception, foreign_key: :user_id
  has_one :user_profile_record, class_name: "UserProfileRecord", dependent: :restrict_with_exception, foreign_key: :user_id
  has_one :profile_record, class_name: "ProfileRecord", through: :user_profile_record

  has_secure_password

  validates :email, presence: true, uniqueness: true

  sig { returns(ActorRecord) }
  def first_actor
    actor_records.find_by!(profile_id: profile_record.id)
  end
end
