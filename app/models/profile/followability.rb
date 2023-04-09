# typed: true
# frozen_string_literal: true

class Profile::Followability
  extend T::Sig

  sig { params(source_profile: Profile).void }
  def initialize(source_profile:)
    @source_profile = source_profile
  end

  sig { params(target_profile: Profile).returns(Profile) }
  def follow(target_profile:)
    return source_profile if source_profile.me?(target_profile:)

    follow = source_profile.follows.where(target_profile:).first_or_initialize(followed_at: Time.current)
    follow.save!

    source_profile
  end

  sig { params(target_profile: Profile).returns(Profile) }
  def unfollow(target_profile:)
    return source_profile if source_profile.me?(target_profile:)

    follow = source_profile.follows.find_by(target_profile:)
    follow&.destroy!

    source_profile
  end

  private

  sig { returns(Profile) }
  attr_reader :source_profile
end
