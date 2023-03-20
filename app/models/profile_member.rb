# typed: strict
# frozen_string_literal: true

class ProfileMember < ApplicationRecord
  belongs_to :account
  belongs_to :profile
end
