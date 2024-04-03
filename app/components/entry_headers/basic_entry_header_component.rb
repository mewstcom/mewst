# typed: strict
# frozen_string_literal: true

class EntryHeaders::BasicEntryHeaderComponent < ApplicationComponent
  sig { params(profile: Profile, time: ActiveSupport::TimeWithZone, detail_path: T.nilable(String)).void }
  def initialize(profile:, time:, detail_path: nil)
    @profile = profile
    @time = time
    @detail_path = detail_path
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
end
