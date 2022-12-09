# typed: strict
# frozen_string_literal: true

class Profile::Friendship
  extend T::Sig

  include ActiveModel::Model

  validate :target_profile_difference

  sig { params(source_profile: Profile, target_profile: Profile).void }
  def initialize(source_profile:, target_profile:)
    @source_profile = source_profile
    @target_profile = target_profile
  end

  sig { returns(T.self_type) }
  def follow
    follow = source_profile.follows.where(target_profile:).first_or_initialize

    follow.save!

    self
  end

  sig { returns(T.self_type) }
  def unfollow
    follow = source_profile.follows.find_by(target_profile:)

    follow&.destroy!

    self
  end

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
