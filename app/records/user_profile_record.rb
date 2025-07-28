# typed: strict
# frozen_string_literal: true

class UserProfileRecord < ApplicationRecord
  self.table_name = "user_profiles"

  belongs_to :user
  belongs_to :profile
end