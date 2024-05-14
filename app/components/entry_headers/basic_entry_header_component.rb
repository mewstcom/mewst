# typed: strict
# frozen_string_literal: true

class EntryHeaders::BasicEntryHeaderComponent < ApplicationComponent
  sig { params(profile: Profile, time: ActiveSupport::TimeWithZone, detail_path: T.nilable(String), follow_checker: T.nilable(FollowChecker)).void }
  def initialize(profile:, time:, detail_path: nil, follow_checker: nil)
    @profile = profile
    @time = time
    @detail_path = detail_path
    @follow_checker = follow_checker
  end

  sig { returns(Profile) }
  attr_reader :profile
  private :profile

  sig { returns(ActiveSupport::TimeWithZone) }
  attr_reader :time
  private :time

  sig { returns(T.nilable(String)) }
  attr_reader :detail_path
  private :detail_path

  sig { returns(T.nilable(FollowChecker)) }
  attr_reader :follow_checker
  private :follow_checker
end
