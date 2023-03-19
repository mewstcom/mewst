# typed: strict
# frozen_string_literal: true

class ProfileMember < ApplicationRecord
  belongs_to :profile
  belongs_to :user
end
