# typed: strict
# frozen_string_literal: true

class Resources::Latest::ResponseError < Resources::Latest::Base
  root_key :error, :errors

  attributes :code, :field, :message
end
