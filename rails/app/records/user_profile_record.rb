# typed: strict
# frozen_string_literal: true

class UserProfileRecord < ApplicationRecord
  self.table_name = "user_profiles"

  belongs_to :user_record, class_name: "UserRecord", foreign_key: :user_id
  belongs_to :profile_record, class_name: "ProfileRecord", foreign_key: :profile_id
end
