# typed: strict
# frozen_string_literal: true

class Profile::Entryability
  extend T::Sig

  sig { params(profile: Profile).void }
  def initialize(profile:)
    @profile = profile
  end

  sig { params(comment: T.nilable(String)).returns(Entry) }
  def create_post_entry(comment:)
    entry = profile.entries.create!(entryable: Post.create!(comment:), published_at: Time.current)

    profile.home_timeline.add_entry(entry:)
    FanoutEntryJob.perform_async(entry_id: entry.id)

    entry
  end

  sig { params(entry: Entry).void }
  def delete_entry(entry:)
    ActiveRecord::Base.transaction do
      entry.destroy!
      profile.home_timeline.remove_entry(entry:)
    end
  end

  private

  sig { returns(Profile) }
  attr_reader :profile
end
