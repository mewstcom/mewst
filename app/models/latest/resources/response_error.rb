# typed: strict
# frozen_string_literal: true

class Latest::Resources::ResponseError < Latest::Resources::Base
  root_key :error, :errors

  attributes :code, :field, :message
end
