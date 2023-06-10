# typed: strict
# frozen_string_literal: true

class Trunk::MewstSchema < GraphQL::Schema
  mutation(Trunk::Types::Objects::MutationType)
  query(Trunk::Types::Objects::QueryType)

  # For batch-loading (see https://graphql-ruby.org/dataloader/overview.html)
  use GraphQL::Dataloader

  # connections.add(Profile::HomeTimeline, Trunk::Connections::TimelineConnection)

  default_max_page_size 50

  # GraphQL-Ruby calls this when something goes wrong while running a query:
  def self.type_error(err, context)
    # if err.is_a?(GraphQL::InvalidNullError)
    #   # report to your bug tracker here
    #   return nil
    # end
    super
  end

  # Union and Interface Resolution
  def self.resolve_type(_abstract_type, obj, _ctx)
    case obj
    when CommentedPost
      Trunk::Types::Objects::CommentedPostType
    else
      raise "Unexpected object: #{obj}"
    end
  end

  # Stop validating when it encounters this many errors:
  validate_max_errors(100)

  # Relay-style Object Identification:

  def self.id_from_object(object, type_definition, query_ctx)
    object.to_gid_param
  end

  def self.object_from_id(global_id, query_ctx = {})
    GlobalID.find(global_id)
  end
end
