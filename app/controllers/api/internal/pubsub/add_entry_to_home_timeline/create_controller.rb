# typed: true
# frozen_string_literal: true

class Api::Internal::Pubsub::AddEntryToHomeTimeline::CreateController < ApplicationController
  include Pubsub::Subscribable

  skip_before_action :verify_authenticity_token
  before_action :require_authentication

  sig { returns(T.untyped) }
  def call
    profile = Profile.only_kept.find_by(id: message_data&.dig("profile_id"))
    entry = Entry.find_by(id: message_data&.dig("entry_id"))

    if profile && entry
      profile.home_timeline.add_entry(entry:)
    end

    head :no_content
  end
end
