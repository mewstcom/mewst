# typed: strict
# frozen_string_literal: true

class FollowChecker
  extend T::Sig

  sig { params(profile: T.nilable(ProfileRecord), target_profiles: T.any(ActiveRecord::Relation, T::Array[ProfileRecord])).void }
  def initialize(profile:, target_profiles:)
    @profile = profile
    @target_profiles = target_profiles
  end

  sig { params(target_profile: ProfileRecord).returns(T::Boolean) }
  def followed?(target_profile:)
    !profile.nil? && followed_profile_ids.include?(target_profile.id)
  end

  sig { returns(T::Array[T::Mewst::DatabaseId]) }
  private def followed_profile_ids
    return [] if profile.nil? || target_profiles.empty?

    profile.not_nil!.follow_records.where(target_profile_record: target_profiles).pluck(:target_profile_record_id)
  end

  sig { returns(T.nilable(ProfileRecord)) }
  attr_reader :profile
  private :profile

  sig { returns(T.any(ActiveRecord::Relation, T::Array[ProfileRecord])) }
  attr_reader :target_profiles
  private :target_profiles
end
