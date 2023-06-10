# typed: strict
# frozen_string_literal: true

class Trunk::Types::Objects::ProfileType < Trunk::Types::Objects::Base
  implements GraphQL::Types::Relay::Node

  field :atname, String, null: false
  field :avatar_url, String, null: false
  field :description, String, null: false
  field :name, String, null: false

  field :home_timeline, Trunk::Types::Objects::PostType.connection_type, null: false, max_page_size: 3

  def home_timeline
    home_timeline = Profile::HomeTimeline.new(profile: object)
    Trunk::Connections::HomeTimelineConnection.new(home_timeline)
  end
end
