# typed: strict
# frozen_string_literal: true

class Commands::FollowProfile < Commands::Base
  include ActiveModel::Model
  include Followable

  validate :target_profile_difference

  sig { params(source_profile: Profile, target_profile: Profile).void }
  def initialize(source_profile:, target_profile:)
    @source_profile = source_profile
    @target_profile = target_profile
  end

  sig { returns(T.self_type) }
  def call
    follow = source_profile.follows.where(target_profile:).first_or_initialize

    follow.save!

    self
  end
end
