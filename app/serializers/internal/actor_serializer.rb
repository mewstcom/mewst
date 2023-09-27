# typed: strict
# frozen_string_literal: true

class Internal::ActorSerializer < Internal::ApplicationSerializer
  root_key :actor, :actors

  attributes :id
end
