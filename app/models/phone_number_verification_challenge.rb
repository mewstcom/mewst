# typed: strict
# frozen_string_literal: true

class PhoneNumberVerificationChallenge < ApplicationRecord
  sig { returns(String) }
  def self.generate_confirmation_code
    6.times.map { rand(10) }.join
  end
end
