# typed: true
# frozen_string_literal: true

module Followable
  extend T::Sig
  extend ActiveSupport::Concern

  private

  sig { returns(Profile) }
  attr_reader :source_profile, :target_profile

  sig { returns(T.untyped) }
  def target_profile_difference
    if source_profile == target_profile
      errors.add(:base, :target_profile_difference)
    end
  end
end
