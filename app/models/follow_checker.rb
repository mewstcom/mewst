# typed: strict
# frozen_string_literal: true

class FollowChecker
  extend T::Sig

  T::Sig::WithoutRuntime.sig do
    params(profile: T.nilable(Profile), target_profiles: T.any(Profile::PrivateRelation, T::Array[Profile])).void
  end
  def initialize(profile:, target_profiles:)
    @profile = profile
    @target_profiles = target_profiles
  end

  sig { params(target_profile: Profile).returns(T::Boolean) }
  def followed?(target_profile:)
    !profile.nil? && followed_profile_ids.include?(target_profile.id)
  end

  sig { returns(T::Array[T::Mewst::DatabaseId]) }
  private def followed_profile_ids
    @followed_profile_ids ||= profile.follows.where(target_profile: target_profiles).pluck(:target_profile_id)
  end

  sig { returns(Profile) }
  attr_reader :profile
  private :profile

  T::Sig::WithoutRuntime.sig { returns(T.any(Profile::PrivateRelation, T::Array[Profile])) }
  attr_reader :target_profiles
  private :target_profiles
end
