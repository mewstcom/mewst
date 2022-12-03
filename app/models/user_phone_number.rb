# typed: strict
# frozen_string_literal: true

class UserPhoneNumber < ApplicationRecord
  belongs_to :user
  belongs_to :phone_number
end
